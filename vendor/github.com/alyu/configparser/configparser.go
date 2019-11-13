// Copyright (c) 2013 - Alex Yu <alex@alexyu.se>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package configparser provides a simple parser for reading/writing configuration (INI) files.
//
// Supports reading/writing the INI file format in addition to:
//
//  - Reading/writing duplicate section names (ex: MySQL NDB engine's config.ini)
//  - Options without values (ex: can be used to group a set of hostnames)
//  - Options without a named section (ex: a simple option=value file)
//  - Find sections with regexp pattern matching on section names, ex: dc1.east.webservers where regex is '.webservers'
//  - # or ; as comment delimiter
//  - = or : as value delimiter
//
package configparser

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
)

// Delimiter is the delimiter to be used between section key and values.
var Delimiter = "="

// Configuration represents a configuration file with its sections and options.
type Configuration struct {
	filePath        string                // configuration file
	sections        map[string]*list.List // fully qualified section name as key
	orderedSections []string              // track the order of section names as they are parsed
	mutex           sync.RWMutex
}

// A Section in a configuration.
type Section struct {
	fqn            string
	options        map[string]string
	orderedOptions []string // track the order of the options as they are parsed
	mutex          sync.RWMutex
}

// NewConfiguration returns a new Configuration instance with an empty file path.
func NewConfiguration() *Configuration {
	return newConfiguration("")
}

// Read parses a specified configuration file and returns a Configuration instance.
func Read(filePath string) (*Configuration, error) {
	filePath = path.Clean(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := newConfiguration(filePath)
	activeSection := config.addSection("global")

	scanner := bufio.NewScanner(bufio.NewReader(file))
	for scanner.Scan() {
		// TODO: maybe trim spaces here
		line := scanner.Text()
		if (len(line) < 0) {
			continue
		}			

		if isSection(line) {
			fqn := strings.Trim(line, " []")
			activeSection = config.addSection(fqn)
			continue
		} 
		// save options and comments
		addOption(activeSection, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// Save the Configuration to file. Creates a backup (.bak) if file already exists.
func Save(c *Configuration, filePath string) (err error) {
	c.mutex.Lock()

	err = os.Rename(filePath, filePath+".bak")
	if err != nil {
		if !os.IsNotExist(err) { // fine if the file does not exists
			return err
		}
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer func() {
		err = f.Close()
	}()

	w := bufio.NewWriter(f)
	defer func() {
		err = w.Flush()
	}()
	c.mutex.Unlock()

	s, err := c.AllSections()
	if err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, v := range s {
		w.WriteString(v.String())
	}

	return err
}

// NewSection creates and adds a new Section with the specified name.
func (c *Configuration) NewSection(fqn string) *Section {
	return c.addSection(fqn)
}

// FilePath returns the configuration file path.
func (c *Configuration) FilePath() string {
	return c.filePath
}

// SetFilePath sets the Configuration file path.
func (c *Configuration) SetFilePath(filePath string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.filePath = filePath
}

// StringValue returns the string value for the specified section and option.
func (c *Configuration) StringValue(section, option string) (value string, err error) {
	s, err := c.Section(section)
	if err != nil {
		return
	}
	value = s.ValueOf(option)
	return
}

// Delete deletes the specified sections matched by a regex name and returns the deleted sections.
func (c *Configuration) Delete(regex string) (sections []*Section, err error) {
	sections, err = c.Find(regex)
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err == nil {
		for _, s := range sections {
			delete(c.sections, s.fqn)
		}
		// remove also from ordered list
		var matched bool
		for i := len(c.orderedSections) - 1; i >= 0; i-- {
			if matched, err = regexp.MatchString(regex, c.orderedSections[i]); matched {
				c.orderedSections = append(c.orderedSections[:i], c.orderedSections[i+1:]...)
			} else {
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return sections, err
}

// Section returns the first section matching the fully qualified section name.
func (c *Configuration) Section(fqn string) (*Section, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if l, ok := c.sections[fqn]; ok {
		for e := l.Front(); e != nil; e = e.Next() {
			s := e.Value.(*Section)
			return s, nil
		}
	}
	return nil, errors.New("Unable to find " + fqn)
}

// AllSections returns a slice of all sections available.
func (c *Configuration) AllSections() ([]*Section, error) {
	return c.Sections("")
}

// Sections returns a slice of Sections matching the fully qualified section name.
func (c *Configuration) Sections(fqn string) ([]*Section, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var sections []*Section

	f := func(lst *list.List) {
		for e := lst.Front(); e != nil; e = e.Next() {
			s := e.Value.(*Section)
			sections = append(sections, s)
		}
	}

	if fqn == "" {
		// Get all sections.
		for _, fqn := range c.orderedSections {
			if lst, ok := c.sections[fqn]; ok {
				f(lst)
			}
		}
	} else {
		if lst, ok := c.sections[fqn]; ok {
			f(lst)
		} else {
			return nil, errors.New("Unable to find " + fqn)
		}
	}

	return sections, nil
}

// Find returns a slice of Sections matching the regexp against the section name.
func (c *Configuration) Find(regex string) ([]*Section, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var sections []*Section
	for key, lst := range c.sections {
		if matched, err := regexp.MatchString(regex, key); matched {
			for e := lst.Front(); e != nil; e = e.Next() {
				s := e.Value.(*Section)
				sections = append(sections, s)
			}
		} else {
			if err != nil {
				return nil, err
			}
		}
	}
	return sections, nil
}

// PrintSection prints a text representation of all sections matching the fully qualified section name.
func (c *Configuration) PrintSection(fqn string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	sections, err := c.Sections(fqn)
	if err == nil {
		for _, section := range sections {
			fmt.Print(section)
		}
	} else {
		fmt.Printf("Unable to find section %v\n", err)
	}
}

// String returns the text representation of a parsed configuration file.
func (c *Configuration) String() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var parts []string
	for _, fqn := range c.orderedSections {
		sections, _ := c.Sections(fqn)
		for _, section := range sections {
			parts = append(parts, section.String())
		}
	}
	return strings.Join(parts, "")
}

// Name returns the name of the section
func (s *Section) Name() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.fqn
}

// Exists returns true if the option exists
func (s *Section) Exists(option string) (ok bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, ok = s.options[option]
	return
}

// ValueOf returns the value of specified option.
func (s *Section) ValueOf(option string) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.options[option]
}

// SetValueFor sets the value for the specified option and returns the old value.
func (s *Section) SetValueFor(option string, value string) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var oldValue string
	oldValue, s.options[option] = s.options[option], value

	return oldValue
}

// Add adds a new option to the section. Adding and existing option will overwrite the old one.
// The old value is returned
func (s *Section) Add(option string, value string) (oldValue string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var ok bool
	if oldValue, ok = s.options[option]; !ok {
		s.orderedOptions = append(s.orderedOptions, option)
	}
	s.options[option] = value

	return oldValue
}

// Delete removes the specified option from the section and returns the deleted option's value.
func (s *Section) Delete(option string) (value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	value = s.options[option]
	delete(s.options, option)
	for i, opt := range s.orderedOptions {
		if opt == option {
			s.orderedOptions = append(s.orderedOptions[:i], s.orderedOptions[i+1:]...)
		}
	}
	return value
}

// Options returns a map of options for the section.
func (s *Section) Options() map[string]string {
	return s.options
}

// OptionNames returns a slice of option names in the same order as they were parsed.
func (s *Section) OptionNames() []string {
	return s.orderedOptions
}

// String returns the text representation of a section with its options.
func (s *Section) String() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var parts []string
	sName := "[" + s.fqn + "]\n"
	if s.fqn == "global" {
		sName = ""
	}
	parts = append(parts, sName)

	for _, opt := range s.orderedOptions {
		value := s.options[opt]
		if value != "" {
			parts = append(parts, opt, Delimiter, value, "\n")
		} else {
			parts = append(parts, opt, "\n")
		}
	}

	return strings.Join(parts, "")
}

//
// Private
//

// newConfiguration creates a new Configuration instance.
func newConfiguration(filePath string) *Configuration {
	return &Configuration{
		filePath: filePath,
		sections: make(map[string]*list.List),
	}
}

func isSection(section string) bool {
	return strings.HasPrefix(section, "[")
}

func addOption(s *Section, option string) {
	var opt, value string
	if opt, value = parseOption(option); value != "" {
		s.options[opt] = value
	} else {
		// only insert keys. ex list of hosts
		s.options[opt] = ""
	}

	s.orderedOptions = append(s.orderedOptions, opt)
}

func parseOption(option string) (opt, value string) {

	split := func(i int, delim string) (opt, value string) {
		// strings.Split cannot handle wsrep_provider_options settings
		opt = strings.Trim(option[:i], " ")
		value = strings.Trim(option[i+1:], " ")
		return
	}

	if i := strings.Index(option, "="); i != -1 {
		opt, value = split(i, "=")
	} else if i := strings.Index(option, ":"); i != -1 {
		opt, value = split(i, ":")
	} else {
		opt = option
	}
	return
}

func (c *Configuration) addSection(fqn string) *Section {
	section := &Section{fqn: fqn, options: make(map[string]string)}

	var lst *list.List
	if lst = c.sections[fqn]; lst == nil {
		lst = list.New()
		c.sections[fqn] = lst
		c.orderedSections = append(c.orderedSections, fqn)
	}

	lst.PushBack(section)

	return section
}
