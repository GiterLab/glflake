# glflake

A distributed unique ID generator inspired by Twitter's Snowflake

## ID Format

    +------------------------------------------------------------------------+
    | 1 Bit Unused | 39 Bit Timestamp | 16 Bit MachineID | 8 Bit Sequence ID |
    +------------------------------------------------------------------------+

    39 bits for time in units of 10 msec (178 years)
    16 bits for a machine id (65536 nodes, distributed machines)
     8 bits for a sequence number (0 ~ 255)

## Install

    go get github.com/GiterLab/glflake

## Usage

    package main

    import (
        "fmt"
        "os"

        "github.com/GiterLab/glflake"
    )

    func main() {
        s := glflake.Settings{}
        // s.Init(0) // set mID, or not Init use default, Default MachineID returns the lower 16 bits of the private IP address.
        dxyid := glflake.NewGlflake(s)

        id, err := dxyid.NextID()
        if err != nil {
            fmt.Println(err)
            os.Exit(0)
        }
        fmt.Println(id, id.LeadingZerosString(19), glflake.Decompose(id))
        idBase64 := id.Base64()
        id, err = glflake.ParseBase64(idBase64)
        if err != nil {
            fmt.Println(err)
            os.Exit(0)
        }
        fmt.Println(idBase64, "-->", id)

        // 19 MAX
        fmt.Println("9223372036854775807", glflake.Decompose(glflake.ID(9223372036854775807))) // 178 years
    }

    // Output:
    //
    // 1931386430720256 0001931386430720256 map[id:1931386430720256 machine-id:8329 msb:0 sequence:0 time:115119602]
    // MTkzMTM4NjQzMDcyMDI1Ng== --> 1931386430720256
    // 9223372036854775807 map[id:9223372036854775807 machine-id:65535 msb:0 sequence:255 time:549755813887]

## License

The MIT License (MIT)

See [LICENSE](https://github.com/GiterLab/dxyflake/blob/master/LICENSE) for details.

## Reference

- [Snowflake](https://github.com/bwmarrin/snowflake)
- [Sonyflake](https://github.com/sony/sonyflake)
