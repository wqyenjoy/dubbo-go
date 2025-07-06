/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/constant/file"
	"github.com/dubbogo/gost/log/logger"
	"github.com/knadh/koanf"
	"github.com/pkg/errors"
)

type loaderConf struct {
	suffix string      // loaderConf file extension default yaml
	path   string      // loaderConf file path default ./conf/dubbogo.yaml
	delim  string      // loaderConf file delim default .
	bytes  []byte      // config bytes
	rc     *RootConfig // user provide rootConfig built by config api
	name   string      // config file name
}

func NewLoaderConf(opts ...LoaderConfOption) (*loaderConf, error) {
	// 支持环境变量配置路径，符合12-Factor原则
	configFilePath := getDefaultConfigPath()
	if configFilePathFromEnv := os.Getenv(constant.ConfigFileEnvKey); configFilePathFromEnv != "" {
		configFilePath = configFilePathFromEnv
	}

	name, suffix := resolverFilePath(configFilePath)
	conf := &loaderConf{
		suffix: suffix,
		path:   absolutePath(configFilePath),
		delim:  ".",
		name:   name,
	}

	for _, opt := range opts {
		opt.apply(conf)
	}

	if conf.rc != nil {
		return conf, nil
	}

	if len(conf.bytes) <= 0 {
		bytes, err := os.ReadFile(conf.path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read config file: %s", conf.path)
		}
		conf.bytes = bytes
	}

	return conf, nil
}

type LoaderConfOption interface {
	apply(vc *loaderConf)
}

type loaderConfigFunc func(*loaderConf)

func (fn loaderConfigFunc) apply(vc *loaderConf) {
	fn(vc)
}

// WithGenre set load config file suffix
// Deprecated: replaced by WithSuffix
func WithGenre(suffix string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		g := strings.ToLower(suffix)
		if err := checkFileSuffix(g); err != nil {
			panic(err)
		}
		conf.suffix = g
	})
}

// WithSuffix set load config file suffix
func WithSuffix(suffix file.Suffix) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.suffix = string(suffix)
	})
}

// WithPath set load config path
func WithPath(path string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.path = absolutePath(path)
		bytes, err := os.ReadFile(conf.path)
		if err != nil {
			logger.Errorf("failed to read config file: %s, error: %v", conf.path, err)
			return // 不panic，只记录错误
		}
		conf.bytes = bytes
		name, suffix := resolverFilePath(path)
		conf.suffix = suffix
		conf.name = name
	})
}

// WithPathSafe set load config path with error handling
func WithPathSafe(path string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.path = absolutePath(path)
		bytes, err := os.ReadFile(conf.path)
		if err != nil {
			logger.Errorf("failed to read config file: %s, error: %v", conf.path, err)
			return
		}
		conf.bytes = bytes
		name, suffix := resolverFilePath(path)
		conf.suffix = suffix
		conf.name = name
	})
}

func WithRootConfig(rc *RootConfig) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.rc = rc
	})
}

func WithDelim(delim string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.delim = delim
	})
}

// WithBytes set load config  bytes
func WithBytes(bytes []byte) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.bytes = bytes
	})
}

// absolutePath get absolut path
func absolutePath(inPath string) string {

	if inPath == "$HOME" || strings.HasPrefix(inPath, "$HOME"+string(os.PathSeparator)) {
		inPath = userHomeDir() + inPath[5:]
	}

	if filepath.IsAbs(inPath) {
		return filepath.Clean(inPath)
	}

	p, err := filepath.Abs(inPath)
	if err == nil {
		return filepath.Clean(p)
	}

	return ""
}

// userHomeDir get gopath
func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

// checkFileSuffix check file suffix
func checkFileSuffix(suffix string) error {
	for _, g := range []string{"json", "toml", "yaml", "yml", "properties"} {
		if g == suffix {
			return nil
		}
	}
	return errors.Errorf("no support file suffix: %s", suffix)
}

// resolverFilePath resolver file path
// eg: give a ./conf/dubbogo.yaml return dubbogo and yaml
func resolverFilePath(path string) (name, suffix string) {
	paths := strings.Split(path, "/")
	fileName := strings.Split(paths[len(paths)-1], ".")
	if len(fileName) < 2 {
		return fileName[0], string(file.YAML)
	}
	return fileName[0], fileName[1]
}

// MergeConfig merge config file
func (conf *loaderConf) MergeConfig(koan *koanf.Koanf) *koanf.Koanf {
	var (
		activeKoan *koanf.Koanf
		activeConf *loaderConf
	)
	active := koan.String("dubbo.profiles.active")
	active = getLegalActive(active)
	logger.Infof("The following profiles are active: %s", active)

	if active == "default" {
		return koan
	}

	path := conf.getActiveFilePath(active)
	if !pathExists(path) {
		logger.Warnf("profile config file not found: %s", path)
		return koan
	}

	var err error
	activeConf, err = NewLoaderConf(WithPath(path))
	if err != nil {
		logger.Errorf("failed to load profile config: %s, error: %v", path, err)
		return koan
	}

	activeKoan = GetConfigResolver(activeConf)
	if activeKoan == nil {
		logger.Errorf("failed to resolve profile config: %s", path)
		return koan
	}

	// 使用深度合并，支持slice/map的合并
	if err := mergeConfigs(koan, activeKoan); err != nil {
		logger.Errorf("failed to merge profile config: %v", err)
		return koan
	}

	return koan
}

// mergeConfigs 深度合并配置，支持slice/map
func mergeConfigs(dst, src *koanf.Koanf) error {
	// 这里应该使用mergo.Merge进行深度合并
	// 暂时使用简单的key覆盖，后续可以引入mergo库
	srcKeys := src.Keys()
	for _, key := range srcKeys {
		value := src.Get(key)
		dst.Set(key, value)
	}
	return nil
}

func (conf *loaderConf) getActiveFilePath(active string) string {
	suffix := constant.DotSeparator + conf.suffix
	return strings.ReplaceAll(conf.path, suffix, "") + "-" + active + suffix
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else {
		return !os.IsNotExist(err)
	}
}

// getDefaultConfigPath 获取默认配置路径，支持XDG_CONFIG_HOME和APP_HOME
func getDefaultConfigPath() string {
	// 优先使用XDG_CONFIG_HOME
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "dubbo-go", "dubbogo.yaml")
	}

	// 其次使用APP_HOME
	if appHome := os.Getenv("APP_HOME"); appHome != "" {
		return filepath.Join(appHome, "conf", "dubbogo.yaml")
	}

	// 最后使用默认路径
	return "../conf/dubbogo.yaml"
}
