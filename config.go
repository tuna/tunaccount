package main

import (
	"errors"
	"os"
	"reflect"
	"strconv"

	"github.com/BurntSushi/toml"
)

type dbBackendEnum uint8

const (
	// DBEnumMongo reporesents mongodb backend
	DBEnumMongo dbBackendEnum = iota
)

func (b *dbBackendEnum) UnmarshalText(text []byte) error {
	s := string(text)
	switch s {
	case `mongo`, `mongodb`:
		*b = DBEnumMongo
	default:
		return errors.New("Invalid value to database backend")
	}
	return nil
}

// A DaemonConfig represents configurations for tunasccount daemon
type DaemonConfig struct {
	DB   DatabaseConfig `toml:"database"`
	LDAP LDAPConfig     `toml:"ldap"`
	HTTP HTTPConfig     `toml:"http"`
}

// A DatabaseConfig is the database config for tunaccount daemon
type DatabaseConfig struct {
	Backend  dbBackendEnum     `toml:"backend"`
	Addr     string            `toml:"addr" default:"127.0.0.1"`
	Port     string            `toml:"port" default:"27017"`
	Name     string            `toml:"name" default:"tunaccount"`
	User     string            `toml:"user"`
	Password string            `toml:"password"`
	Options  map[string]string `toml:"options"`
}

// An LDAPConfig is ldap server configs
type LDAPConfig struct {
	ListenAddr string `toml:"listen_addr" default:"127.0.0.1"`
	ListenPort int    `toml:"listen_port" default:"389"`
}

// An HTTPConfig is http server configs
type HTTPConfig struct {
	ListenAddr string `toml:"listen_addr" default:"127.0.0.1"`
	ListenPort int    `toml:"listen_port" default:"9501"`
}

func newDefaultConfig() *DaemonConfig {
	cfg := new(DaemonConfig)
	var setDefault func(reflect.Value)
	setDefault = func(v reflect.Value) {
		for i := 0; i < v.NumField(); i++ {
			vf := v.Field(i)
			if vf.Kind() == reflect.Struct {
				setDefault(vf)
				continue
			}

			dv := v.Type().Field(i).Tag.Get("default")
			if dv != "" && vf.CanSet() {
				switch vf.Kind() {
				case reflect.String:
					vf.SetString(dv)
				case reflect.Int:
					iv, _ := strconv.Atoi(dv)
					vf.SetInt(int64(iv))
				}
			}
		}
	}

	setDefault(reflect.ValueOf(cfg).Elem())
	return cfg
}

func loadConfig(cfgFile string) (*DaemonConfig, error) {
	cfg := newDefaultConfig()

	if _, err := os.Stat(cfgFile); err == nil {
		if _, err := toml.DecodeFile(cfgFile, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
