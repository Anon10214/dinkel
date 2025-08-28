/*
Package config manages fuzzing configs for cobra-cli commands.

This package provides the ability to set a default fuzzing config
that gets used by the fuzzing commands.

It additionally simplifies reading in bug reports.
*/
package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/apacheage"
	"github.com/Anon10214/dinkel/models/falkordb"
	"github.com/Anon10214/dinkel/models/memgraph"
	"github.com/Anon10214/dinkel/models/neo4j"
	"github.com/Anon10214/dinkel/models/redisgraph"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/Anon10214/dinkel/scheduler/strategy"
	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

type targetConfig map[string]struct {
	IgnoredErrorMessages  []string `yaml:"ignoredErrors"`
	ReportedErrorMessages []string `yaml:"reportedErrors"`
	BugReportTemplate     string   `yaml:"bugreportTemplate"`
}

var defaultConfig scheduler.Config

// SetDefaultConfig can be used to set the default fuzzing config to be used by the cobra-cli commands
func SetDefaultConfig(conf scheduler.Config) {
	defaultConfig = conf
}

// BugReport represents a bug report parsed from a generated .yml file by the fuzzer
type BugReport struct {
	FilePath           string                   `yaml:"-"`
	ReportName         string                   `yaml:"-"` // The base of the file path, without any file extension
	Strategy           strategy.Strategy        `yaml:"-"`
	Target             string                   `yaml:"target"`
	StrategyNum        strategy.FuzzingStrategy `yaml:"strategy"`
	TimeFound          string                   `yaml:"time_found"`
	OffendingCommit    string                   `yaml:"offending_commit"`
	ByteStringAsString string                   `yaml:"byte_string"`
	Query              []string                 `yaml:"query"`
	ByteString         []byte
}

// ReadBugreport reads in the bugreport pointed to by the given path
func ReadBugreport(filePath string) (*BugReport, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Join(errors.New("failed to read passed bugreport - %v"), err)
	}
	curBugreport := BugReport{}
	if err := yaml.Unmarshal(data, &curBugreport); err != nil {
		return nil, errors.Join(errors.New("failed to unmarshal bugreport - %v"), err)
	}

	curBugreport.FilePath = strings.TrimSuffix(filePath, ".yml")
	curBugreport.ReportName = path.Base(curBugreport.FilePath)

	curBugreport.ByteString, err = base64.StdEncoding.DecodeString(curBugreport.ByteStringAsString)
	if err != nil {
		return nil, errors.Join(errors.New("invalid byte string in bugreport - %v"), err)
	}

	curBugreport.Strategy = curBugreport.StrategyNum.ToStrategy()

	return &curBugreport, nil
}

// GetConfigForTarget returns the fuzzing config associated with a given fuzzing target
func GetConfigForTarget(target string, configPath string) (scheduler.Config, error) {
	conf := defaultConfig

	// Read in and parse the target config for a given target
	data, err := os.ReadFile(configPath)
	if err != nil {
		return scheduler.Config{}, errors.Join(errors.New("failed to read passed target config - "), err)
	}
	targetConf := targetConfig{}
	if err := yaml.Unmarshal(data, &targetConf); err != nil {
		return scheduler.Config{}, errors.Join(errors.New("failed to unmarshal target config - "), err)
	}

	if _, ok := targetConf[target]; !ok {
		return scheduler.Config{}, fmt.Errorf("target %s not found in target config at %s", target, configPath)
	}

	// Create the ignored error message regexp
	curTargetConf := targetConf[target]
	ignoredErrorMessages := ""
	for _, msg := range curTargetConf.IgnoredErrorMessages {
		ignoredErrorMessages += fmt.Sprintf("(%s)|", msg)
	}
	ignoredErrorMessages = ignoredErrorMessages[:len(ignoredErrorMessages)-1]
	ignoredErrorMessagesRegexp, err := regexp.Compile(ignoredErrorMessages)
	if err != nil {
		return scheduler.Config{}, errors.Join(errors.New("failed to read the regexp for ignored error messages - "), err)
	}

	// Create the reported error message regexp
	reportedErrorMessages := ""
	for _, msg := range curTargetConf.ReportedErrorMessages {
		reportedErrorMessages += fmt.Sprintf("(%s)|", msg)
	}
	reportedErrorMessages = reportedErrorMessages[:len(reportedErrorMessages)-1]
	reportedErrorMessagesRegexp, err := regexp.Compile(reportedErrorMessages)
	if err != nil {
		return scheduler.Config{}, errors.Join(errors.New("failed to read the regexp for reported error messages - "), err)
	}

	errorMessageRegex := dbms.ErrorMessageRegex{
		Ignored:  ignoredErrorMessagesRegexp,
		Reported: reportedErrorMessagesRegexp,
	}

	conf.ErrorMessageRegex = &errorMessageRegex
	if conf.BugReportTemplate, err = template.New("bugreport-markdown").Funcs(sprig.FuncMap()).Parse(curTargetConf.BugReportTemplate); err != nil {
		return scheduler.Config{}, errors.Join(errors.New("failed to parse the bug report template - "), err)
	}

	// Make sure the template can generate a bug report
	if err := conf.BugReportTemplate.Execute(&bytes.Buffer{}, scheduler.BugreportMarkdownData{}); err != nil {
		return scheduler.Config{}, errors.Join(errors.New("failed to execute the bug report template, is the template valid? - "), err)
	}

	conf.TargetDB = target

	// Set the fuzz target
	switch target {
	case "neo4j":
		conf.DB = &neo4j.Driver{}
		conf.Implementation = neo4j.Implementation{}
	case "redisgraph":
		conf.DB = &redisgraph.Driver{}
		conf.Implementation = redisgraph.Implementation{}
	case "falkordb":
		conf.DB = &falkordb.Driver{}
		conf.Implementation = falkordb.Implementation{}
	case "memgraph":
		conf.DB = &memgraph.Driver{}
		conf.Implementation = memgraph.Implementation{}
	case "apache-age":
		conf.DB = &apacheage.Driver{}
		conf.Implementation = apacheage.Implementation{}
	default:
		return conf, errors.New("invalid target")
	}
	return conf, nil
}
