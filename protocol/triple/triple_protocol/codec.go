// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	hessian "github.com/apache/dubbo-go-hessian2"

	perrors "github.com/pkg/errors"

	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/interoperability"
	"github.com/dubbogo/gost/log/logger"
	msgpack "github.com/ugorji/go/codec"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoiface"
)

const (
	codecNameProto           = "proto"
	codecNameJSON            = "json"
	codecNameHessian2        = "hessian2"
	codecNameMsgPack         = "msgpack"
	codecNameJSONCharsetUTF8 = codecNameJSON + "; charset=utf-8"
)

// Codec marshals structs (typically generated from a schema) to and from bytes.
type Codec interface {
	// Name returns the name of the Codec.
	//
	// This may be used as part of the Content-Type within HTTP. For example,
	// with gRPC this is the content subtype, so "application/grpc+proto" will
	// map to the Codec with name "proto".
	//
	// Names must not be empty.
	Name() string
	// Marshal marshals the given message.
	//
	// Marshal may expect a specific type of message, and will error if this type
	// is not given.
	Marshal(any) ([]byte, error)
	// Unmarshal unmarshals the given message.
	//
	// Unmarshal may expect a specific type of message, and will error if this
	// type is not given.
	Unmarshal([]byte, any) error
}

// stableCodec is an extension to Codec for serializing with stable output.
type stableCodec interface {
	Codec

	// MarshalStable marshals the given message with stable field ordering.
	//
	// MarshalStable should return the same output for a given input. Although
	// it is not guaranteed to be canonicalized, the marshaling routine for
	// MarshalStable will opt for the most normalized output available for a
	// given serialization.
	//
	// For practical reasons, it is possible for MarshalStable to return two
	// different results for two inputs considered to be "equal" in their own
	// domain, and it may change in the future with codec updates, but for
	// any given concrete value and any given version, it should return the
	// same output.
	MarshalStable(any) ([]byte, error)

	// IsBinary returns true if the marshaled data is binary for this codec.
	//
	// If this function returns false, the data returned from Marshal and
	// MarshalStable are considered valid text and may be used in contexts
	// where text is expected.
	IsBinary() bool
}

type protoBinaryCodec struct{}

var _ Codec = (*protoBinaryCodec)(nil)

func (c *protoBinaryCodec) Name() string { return codecNameProto }

func (c *protoBinaryCodec) Marshal(message any) ([]byte, error) {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, errNotProto(message)
	}
	return proto.Marshal(protoMessage)
}

func (c *protoBinaryCodec) Unmarshal(data []byte, message any) error {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return errNotProto(message)
	}
	return proto.Unmarshal(data, protoMessage)
}

func (c *protoBinaryCodec) MarshalStable(message any) ([]byte, error) {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, errNotProto(message)
	}
	// protobuf does not offer a canonical output today, so this format is not
	// guaranteed to match deterministic output from other protobuf libraries.
	// In addition, unknown fields may cause inconsistent output for otherwise
	// equal messages.
	// https://github.com/golang/protobuf/issues/1121
	options := proto.MarshalOptions{Deterministic: true}
	return options.Marshal(protoMessage)
}

func (c *protoBinaryCodec) IsBinary() bool {
	return true
}

type protoJSONCodec struct {
	name string
}

var _ Codec = (*protoJSONCodec)(nil)

func (c *protoJSONCodec) Name() string { return c.name }

func (c *protoJSONCodec) Marshal(message any) ([]byte, error) {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, errNotProto(message)
	}
	var options = protojson.MarshalOptions{
		UseProtoNames: true,
	}
	return options.Marshal(protoMessage)
}

func (c *protoJSONCodec) Unmarshal(binary []byte, message any) error {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return errNotProto(message)
	}
	if len(binary) == 0 {
		return errors.New("zero-length payload is not a valid JSON object")
	}
	var options protojson.UnmarshalOptions
	return options.Unmarshal(binary, protoMessage)
}

func (c *protoJSONCodec) MarshalStable(message any) ([]byte, error) {
	// protojson does not offer a "deterministic" field ordering, but fields
	// are still ordered consistently by their index. However, protojson can
	// output inconsistent whitespace for some reason, therefore it is
	// suggested to use a formatter to ensure consistent formatting.
	// https://github.com/golang/protobuf/issues/1373
	messageJSON, err := c.Marshal(message)
	if err != nil {
		return nil, err
	}
	compactedJSON := bytes.NewBuffer(messageJSON[:0])
	if err = json.Compact(compactedJSON, messageJSON); err != nil {
		return nil, err
	}
	return compactedJSON.Bytes(), nil
}

func (c *protoJSONCodec) IsBinary() bool {
	return false
}

// todo(DMwangnima): add unit tests
type protoWrapperCodec struct {
	innerCodec Codec
}

func (c *protoWrapperCodec) Name() string {
	return c.innerCodec.Name()
}

func (c *protoWrapperCodec) Marshal(message any) ([]byte, error) {
	// debug: log message dynamic type
	logger.Warnf("protoWrapperCodec.Marshal: msg type=%T", message)
	// If the message is a slice of args, wrap as TripleRequestWrapper
	if reqs, ok := message.([]any); ok {
		logger.Warnf("protoWrapperCodec.Marshal: detected []any len=%d", len(reqs))
		for i := range reqs {
			logger.Warnf("protoWrapperCodec.Marshal: arg[%d] type=%T", i, reqs[i])
		}
		// flatten nested single-arg slice: []any{ []any{...} } → []any{...}
		if len(reqs) == 1 {
			if inner, ok2 := reqs[0].([]any); ok2 {
				logger.Warnf("protoWrapperCodec.Marshal: flatten nested args len=%d", len(inner))
				reqs = inner
			} else {
				// also support []interface{} concrete type
				rv := reflect.ValueOf(reqs[0])
				if rv.IsValid() && rv.Kind() == reflect.Slice {
					innerLen := rv.Len()
					flatten := make([]any, innerLen)
					for i := 0; i < innerLen; i++ {
						flatten[i] = rv.Index(i).Interface()
					}
					logger.Warnf("protoWrapperCodec.Marshal: flatten generic slice len=%d", innerLen)
					reqs = flatten
				}
			}
		}
		reqsLen := len(reqs)
		reqsBytes := make([][]byte, reqsLen)
		reqsTypes := make([]string, reqsLen)
		for i, req := range reqs {
			b, err := c.innerCodec.Marshal(req)
			if err != nil {
				return nil, err
			}
			reqsBytes[i] = b
			reqsTypes[i] = getArgType(req)
		}
		wrapperReq := &interoperability.TripleRequestWrapper{
			SerializeType: c.innerCodec.Name(),
			Args:          reqsBytes,
			ArgTypes:      reqsTypes,
		}
		logger.Warnf("protoWrapperCodec.Marshal: request wrapper serialize=%s args=%d", wrapperReq.SerializeType, len(wrapperReq.Args))
		return proto.Marshal(wrapperReq)
	}
	// Also support []interface{} and arbitrary slice kinds by converting to []any
	if ifaces, ok := message.([]interface{}); ok {
		logger.Warnf("protoWrapperCodec.Marshal: detected []interface{} len=%d", len(ifaces))
		reqs := make([]any, len(ifaces))
		for i := range ifaces {
			reqs[i] = ifaces[i]
		}
		reqsLen := len(reqs)
		reqsBytes := make([][]byte, reqsLen)
		reqsTypes := make([]string, reqsLen)
		for i, req := range reqs {
			b, err := c.innerCodec.Marshal(req)
			if err != nil {
				return nil, err
			}
			reqsBytes[i] = b
			reqsTypes[i] = getArgType(req)
		}
		wrapperReq := &interoperability.TripleRequestWrapper{
			SerializeType: c.innerCodec.Name(),
			Args:          reqsBytes,
			ArgTypes:      reqsTypes,
		}
		logger.Warnf("protoWrapperCodec.Marshal: request wrapper ([]interface{}) serialize=%s args=%d", wrapperReq.SerializeType, len(wrapperReq.Args))
		return proto.Marshal(wrapperReq)
	}
	if rv := reflect.ValueOf(message); rv.IsValid() && rv.Kind() == reflect.Slice {
		logger.Warnf("protoWrapperCodec.Marshal: detected reflect slice len=%d elem=%s", rv.Len(), rv.Type().Elem().String())
		l := rv.Len()
		reqs := make([]any, l)
		for i := 0; i < l; i++ {
			reqs[i] = rv.Index(i).Interface()
		}
		reqsLen := len(reqs)
		reqsBytes := make([][]byte, reqsLen)
		reqsTypes := make([]string, reqsLen)
		for i, req := range reqs {
			b, err := c.innerCodec.Marshal(req)
			if err != nil {
				return nil, err
			}
			reqsBytes[i] = b
			reqsTypes[i] = getArgType(req)
		}
		wrapperReq := &interoperability.TripleRequestWrapper{
			SerializeType: c.innerCodec.Name(),
			Args:          reqsBytes,
			ArgTypes:      reqsTypes,
		}
		logger.Warnf("protoWrapperCodec.Marshal: request wrapper (reflect slice) serialize=%s args=%d", wrapperReq.SerializeType, len(wrapperReq.Args))
		return proto.Marshal(wrapperReq)
	}
	// Force-wrap fallback: if message looks like a slice but above branches missed, never send Args=0
	if arr, ok := message.([]interface{}); ok && len(arr) == 0 {
		logger.Warnf("protoWrapperCodec.Marshal: force-wrap empty []interface{} to avoid Args=0")
		wrapperReq := &interoperability.TripleRequestWrapper{
			SerializeType: c.innerCodec.Name(),
			Args:          [][]byte{},
			ArgTypes:      []string{},
		}
		return proto.Marshal(wrapperReq)
	}
	// Otherwise, wrap as TripleResponseWrapper
	b, err := c.innerCodec.Marshal(message)
	if err != nil {
		return nil, err
	}
	wrapperResp := &interoperability.TripleResponseWrapper{
		SerializeType: c.innerCodec.Name(),
		Data:          b,
		Type:          getArgType(message),
	}
	return proto.Marshal(wrapperResp)
}

func (c *protoWrapperCodec) Unmarshal(binary []byte, message any) error {
	// Try RequestWrapper first (request path)
	var params []any
	isParamsSlice := false
	if ps, ok := message.([]any); ok {
		params = ps
		isParamsSlice = true
	} else {
		params = []any{message}
	}

	// Try RequestWrapper
	var wrapperReq interoperability.TripleRequestWrapper
	if err := proto.Unmarshal(binary, &wrapperReq); err == nil {
		logger.Warnf("protoWrapperCodec.Unmarshal: request wrapper parsed, serialize=%s, args=%d, expect=%d", wrapperReq.SerializeType, len(wrapperReq.Args), len(params))
		if len(wrapperReq.Args) == len(params) && len(wrapperReq.Args) > 0 {
			for i, arg := range wrapperReq.Args {
				logger.Warnf("protoWrapperCodec.Unmarshal: decoding arg[%d] into %T", i, params[i])
				if err := c.innerCodec.Unmarshal(arg, params[i]); err != nil {
					return err
				}
			}
			return nil
		}
	}

	// If target is []any (request), skip ResponseWrapper branch to avoid misrouting
	if isParamsSlice {
		// skip response wrapper path
	} else {
		// Defensive: try ResponseWrapper (response path)
		var resp interoperability.TripleResponseWrapper
		if err := proto.Unmarshal(binary, &resp); err == nil && len(resp.Data) > 0 {
			// If inner codec is hessian2 and target is []any, decode Data as list/seq
			if _, ok := c.innerCodec.(*hessian2Codec); ok {
				if ps, isSlice := message.([]any); isSlice {
					dec := hessian.NewDecoder(resp.Data)
					first, err := dec.Decode()
					if err != nil {
						return err
					}
					if rv := reflect.ValueOf(first); rv.IsValid() && rv.Kind() == reflect.Slice && rv.Len() == len(ps) {
						for i := 0; i < rv.Len(); i++ {
							if err := reflectResponse(rv.Index(i).Interface(), ps[i]); err != nil {
								return err
							}
						}
						return nil
					}
					if len(ps) > 0 {
						if err := reflectResponse(first, ps[0]); err != nil {
							return err
						}
						for i := 1; i < len(ps); i++ {
							val, err := dec.Decode()
							if err != nil {
								return err
							}
							if err := reflectResponse(val, ps[i]); err != nil {
								return err
							}
						}
						return nil
					}
				}
				// Not a []any target: just decode inner data normally
				return c.innerCodec.Unmarshal(resp.Data, message)
			}
			return c.innerCodec.Unmarshal(resp.Data, message)
		}
	}

	// Fallback: Non-wrapper payload with Hessian2; decode sequentially
	if _, ok := c.innerCodec.(*hessian2Codec); ok {
		logger.Warnf("protoWrapperCodec.Unmarshal: hessian2 fallback, params=%d", len(params))
		dec := hessian.NewDecoder(binary)
		first, err := dec.Decode()
		if err != nil {
			logger.Errorf("protoWrapperCodec.Unmarshal: first Decode error=%v", err)
			return err
		}
		logger.Warnf("protoWrapperCodec.Unmarshal: first type=%T", first)
		if rv := reflect.ValueOf(first); rv.IsValid() && rv.Kind() == reflect.Slice {
			logger.Warnf("protoWrapperCodec.Unmarshal: first slice len=%d expect=%d", rv.Len(), len(params))
			if rv.Len() == len(params) {
				for i := 0; i < rv.Len(); i++ {
					if err := reflectResponse(rv.Index(i).Interface(), params[i]); err != nil {
						logger.Errorf("protoWrapperCodec.Unmarshal: slice[%d] reflect error=%v", i, err)
						return err
					}
					logger.Warnf("protoWrapperCodec.Unmarshal: slice[%d] reflect ok", i)
				}
				return nil
			}
		}
		// otherwise, treat first as the single response value
		if len(params) > 0 {
			return reflectResponse(first, params[0])
		}
		return nil
	}
	// Last resort: try inner codec on whole payload
	return c.innerCodec.Unmarshal(binary, message)
}

func newProtoWrapperCodec(innerCodec Codec) *protoWrapperCodec {
	return &protoWrapperCodec{innerCodec: innerCodec}
}

// todo(DMwangnima): add unit tests
type hessian2Codec struct{}

func (h *hessian2Codec) Name() string {
	return codecNameHessian2
}

func (c *hessian2Codec) Marshal(message any) ([]byte, error) {
	encoder := hessian.NewEncoder()
	logger.Warnf("message type: %T", message)
	if err := encoder.Encode(message); err != nil {
		logger.Errorf("err: %v", err)
		return nil, err
	}

	return encoder.Buffer(), nil
}

func (c *hessian2Codec) Unmarshal(binary []byte, message any) error {
	logger.Warnf("binary: %v", binary)
	decoder := hessian.NewDecoder(binary)
	val, err := decoder.Decode()
	if err != nil {
		logger.Errorf("err: %v", err)
		return err
	}
	logger.Warnf("val: %v", val)
	return reflectResponse(val, message)
}

type msgpackCodec struct{}

func (c *msgpackCodec) Name() string {
	return codecNameMsgPack
}

func (c *msgpackCodec) Marshal(message any) ([]byte, error) {
	var out []byte
	encoder := msgpack.NewEncoderBytes(&out, new(msgpack.MsgpackHandle))
	return out, encoder.Encode(message)
}

func (c *msgpackCodec) Unmarshal(binary []byte, message any) error {
	decoder := msgpack.NewDecoderBytes(binary, new(msgpack.MsgpackHandle))
	return decoder.Decode(message)
}

// removed PlainJSONCodec

// readOnlyCodecs is a read-only interface to a map of named codecs.
type readOnlyCodecs interface {
	// Get gets the Codec with the given name.
	Get(string) Codec
	// Protobuf gets the user-supplied protobuf codec, falling back to the default
	// implementation if necessary.
	//
	// This is helpful in the gRPC protocol, where the wire protocol requires
	// marshaling protobuf structs to binary even if the RPC procedures were
	// generated from a different IDL.
	Protobuf() Codec
	// Names returns a copy of the registered codec names. The returned slice is
	// safe for the caller to mutate.
	Names() []string
}

func newReadOnlyCodecs(nameToCodec map[string]Codec) readOnlyCodecs {
	return &codecMap{
		nameToCodec: nameToCodec,
	}
}

type codecMap struct {
	nameToCodec map[string]Codec
}

func (m *codecMap) Get(name string) Codec {
	return m.nameToCodec[name]
}

func (m *codecMap) Protobuf() Codec {
	if pb, ok := m.nameToCodec[codecNameProto]; ok {
		return pb
	}
	return &protoBinaryCodec{}
}

func (m *codecMap) Names() []string {
	names := make([]string, 0, len(m.nameToCodec))
	for name := range m.nameToCodec {
		names = append(names, name)
	}
	return names
}

func errNotProto(message any) error {
	if _, ok := message.(protoiface.MessageV1); ok {
		return fmt.Errorf("%T uses github.com/golang/protobuf, but triple only supports google.golang.org/protobuf: see https://go.dev/blog/protobuf-apiv2", message)
	}
	return fmt.Errorf("%T doesn't implement proto.Message", message)
}

// Definitions from dubbogo/grpc-go
func getArgType(v any) string {
	if v == nil {
		return "V"
	}

	switch v := v.(type) {
	// Serialized tags for base types
	case nil:
		return "V"
	case bool:
		return "boolean"
	case []bool:
		return "[Z"
	case byte:
		return "byte"
	case []byte:
		return "[B"
	case int8:
		return "byte"
	case []int8:
		return "[B"
	case int16:
		return "short"
	case []int16:
		return "[S"
	case uint16: // Equivalent to Char of Java
		return "char"
	case []uint16:
		return "[C"
	case int: // Equivalent to Long of Java
		return "long"
	case []int:
		return "[J"
	case int32:
		return "int"
	case []int32:
		return "[I"
	case int64:
		return "long"
	case []int64:
		return "[J"
	case time.Time:
		return "java.util.Date"
	case []time.Time:
		return "[Ljava.util.Date"
	case float32:
		return "float"
	case []float32:
		return "[F"
	case float64:
		return "double"
	case []float64:
		return "[D"
	case string:
		return "java.lang.String"
	case []string:
		return "[Ljava.lang.String;"
	case []hessian.Object:
		return "[Ljava.lang.Object;"
	case map[any]any:
		// return  "java.util.HashMap"
		return "java.util.Map"
	case hessian.POJOEnum:
		return v.JavaClassName()
	//  Serialized tags for complex types
	default:
		t := reflect.TypeOf(v)
		if reflect.Ptr == t.Kind() {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Struct:
			v, ok := v.(hessian.POJO)
			if ok {
				return v.JavaClassName()
			}
			return "java.lang.Object"
		case reflect.Slice, reflect.Array:
			if t.Elem().Kind() == reflect.Struct {
				return "[Ljava.lang.Object;"
			}
			// return "java.util.ArrayList"
			return "java.util.List"
		case reflect.Map: // Enter here, map may be map[string]int
			return "java.util.Map"
		default:
			return ""
		}
	}
}

func reflectResponse(in any, out any) error {
	if in == nil {
		// 容忍 nil 入参：将 out 置为其零值
		if out != nil {
			ov := reflect.ValueOf(out)
			if ov.IsValid() && ov.Kind() == reflect.Ptr {
				ov = ov.Elem()
				if ov.IsValid() && ov.CanSet() {
					ov.Set(reflect.Zero(ov.Type()))
					return nil
				}
			}
		}
		return nil
	}

	if out == nil {
		return perrors.Errorf("@out is nil")
	}
	if reflect.TypeOf(out).Kind() != reflect.Ptr {
		return perrors.Errorf("@out should be a pointer")
	}

	// Special handling: if out is interface{} whose dynamic value is a non-nil pointer,
	// write directly into the pointed concrete value instead of replacing the interface.
	{
		ov := reflect.ValueOf(out)
		if ov.IsValid() && ov.Kind() == reflect.Ptr {
			inner := ov.Elem()
			if inner.IsValid() && inner.Kind() == reflect.Interface {
				dyn := inner.Elem()
				if dyn.IsValid() && dyn.Kind() == reflect.Ptr && !dyn.IsNil() {
					inVal := hessian.EnsurePackValue(in)
					dest := dyn.Elem()
					hessian.SetValue(dest, inVal)
					return nil
				}
			}
		}
	}

	inValue := hessian.EnsurePackValue(in)
	outValue := hessian.EnsurePackValue(out)

	outType := outValue.Type().String()
	if outType == "interface {}" || outType == "*interface {}" {
		hessian.SetValue(outValue, inValue)
		return nil
	}

	switch inValue.Type().Kind() {
	case reflect.Slice, reflect.Array:
		return copySlice(inValue, outValue)
	case reflect.Map:
		return copyMap(inValue, outValue)
	default:
		hessian.SetValue(outValue, inValue)
	}

	return nil
}

// copySlice copy from inSlice to outSlice
func copySlice(inSlice, outSlice reflect.Value) error {
	if inSlice.IsNil() {
		return perrors.New("@in is nil")
	}
	if inSlice.Kind() != reflect.Slice {
		return perrors.Errorf("@in is not slice, but %v", inSlice.Kind())
	}
	for outSlice.Kind() == reflect.Ptr {
		outSlice = outSlice.Elem()
	}

	// Fallback: target is not a slice, avoid panic and set value directly
	if outSlice.Kind() != reflect.Slice {
		inIface := inSlice.Interface()
		inVal := hessian.EnsurePackValue(inIface)
		hessian.SetValue(outSlice, inVal)
		return nil
	}

	size := inSlice.Len()
	outSlice.Set(reflect.MakeSlice(outSlice.Type(), size, size))

	outElemType := outSlice.Type().Elem()
	for i := 0; i < size; i++ {
		inSliceValue := inSlice.Index(i)
		// Fast path: directly assignable
		if inSliceValue.Type().AssignableTo(outElemType) {
			// Convert to ensure exact type match when assignable
			if inSliceValue.Type() != outElemType && inSliceValue.Type().ConvertibleTo(outElemType) {
				outSlice.Index(i).Set(inSliceValue.Convert(outElemType))
			} else {
				outSlice.Index(i).Set(inSliceValue)
			}
			continue
		}
		// Lenient path: try conversion via hessian helpers
		outElem := reflect.New(outElemType).Elem()
		inPacked := hessian.EnsurePackValue(inSliceValue.Interface())
		hessian.SetValue(outElem, inPacked)
		if outElem.IsValid() && outElem.Type() == outElemType {
			outSlice.Index(i).Set(outElem)
			continue
		}
		// special-case string destination
		if outElemType.Kind() == reflect.String {
			outSlice.Index(i).Set(reflect.ValueOf(fmt.Sprint(inSliceValue.Interface())))
			continue
		}
		return perrors.Errorf("in element type [%s] can not assign to out element type [%s]",
			inSliceValue.Type().String(), outElemType.String())
	}

	return nil
}

// copyMap copy from in map to out map
func copyMap(inMapValue, outMapValue reflect.Value) error {
	if inMapValue.IsNil() {
		return perrors.New("@in is nil")
	}
	if !inMapValue.CanInterface() {
		return perrors.New("@in's Interface can not be used.")
	}
	if inMapValue.Kind() != reflect.Map {
		return perrors.Errorf("@in is not map, but %v", inMapValue.Kind())
	}

	outMapType := hessian.UnpackPtrType(outMapValue.Type())
	hessian.SetValue(outMapValue, reflect.MakeMap(outMapType))

	outKeyType := outMapType.Key()

	outMapValue = hessian.UnpackPtrValue(outMapValue)
	outValueType := outMapValue.Type().Elem()

	for _, inKey := range inMapValue.MapKeys() {
		inValue := inMapValue.MapIndex(inKey)

		if !inKey.Type().AssignableTo(outKeyType) {
			return perrors.Errorf("in Key:{type:%s, value:%#v} can not assign to out Key:{type:%s} ",
				inKey.Type().String(), inKey, outKeyType.String())
		}
		if !inValue.Type().AssignableTo(outValueType) {
			return perrors.Errorf("in Value:{type:%s, value:%#v} can not assign to out value:{type:%s}",
				inValue.Type().String(), inValue, outValueType.String())
		}
		outMapValue.SetMapIndex(inKey, inValue)
	}

	return nil
}
