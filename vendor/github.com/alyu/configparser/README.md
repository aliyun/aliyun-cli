configparser
============
Package configparser provides a simple parser for reading/writing configuration (INI) files.

Supports reading/writing the INI file format in addition to:

- Reading/writing duplicate section names (ex: MySQL NDB engine's config.ini)
- Options without values (ex: can be used to group a set of hostnames)
- Options without a named section (ex: a simple option=value file)
- Find sections with regexp pattern matching on section names, ex: dc1.east.webservers where regex is '.webservers'
- # or ; as comment delimiter
- = or : as value delimiter

```go
package configparser_test

import (
    "fmt"
    "github.com/alyu/configparser"
    "log"
)

// Read and modify a configuration file
func Example() {
    // set a custom delimiter to be used for key/value seperation
    configparser.Delimiter = "="
    
    config, err := configparser.Read("/etc/config.ini")
    if err != nil {
        log.Fatal(err)
    }
    // Print the full configuration
    fmt.Println(config)

    // get a section
    section, err := config.Section("MYSQLD DEFAULT")
    if err != nil {
        log.Fatal(err)
    } else {
        fmt.Printf("TotalSendBufferMemory=%s\n", section.ValueOf("TotalSendBufferMemory"))

        // set new value
        var oldValue = section.SetValueFor("TotalSendBufferMemory", "256M")
        fmt.Printf("TotalSendBufferMemory=%s, old value=%s\n", section.ValueOf("TotalSendBufferMemory"), oldValue)

        // delete option
        oldValue = section.Delete("DefaultOperationRedoProblemAction")
        fmt.Println("Deleted DefaultOperationRedoProblemAction: " + oldValue)

        // add new options
        section.Add("innodb_buffer_pool_size", "64G")
        section.Add("innodb_buffer_pool_instances", "8")
    }

    // add a new section and options
    section = config.NewSection("NDBD MGM")
    section.Add("NodeId", "2")
    section.Add("HostName", "10.10.10.10")
    section.Add("PortNumber", "1186")
    section.Add("ArbitrationRank", "1")

    // find all sections ending with .webservers
    sections, err := config.Find(".webservers$")
    if err != nil {
        log.Fatal(err)
    }
    for _, section := range sections {
        fmt.Print(section)
    }
    // or
    config.PrintSection("dc1.webservers")

    sections, err = config.Delete("NDB_MGMD DEFAULT")
    if err != nil {
        log.Fatal(err)
    }
    // deleted sections
    for _, section := range sections {
        fmt.Print(section)
    }

    options := section.Options()
    fmt.Println(options["HostName"])

    // save the new config. the original will be renamed to /etc/config.ini.bak
    err = configparser.Save(config, "/etc/config.ini")
    if err != nil {
        log.Fatal(err)
    }
}
```