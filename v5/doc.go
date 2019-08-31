/*
Package proj provides an interface to the Cartographic Projections Library PROJ [cartography].

See: https://proj.org/

This package supports PROJ version 5 and above.

For PROJ.4, see: https://github.com/pebbe/go-proj-4

Example usage:

    ctx := proj.NewContext()
    defer ctx.Close() // if omitted, this will be called on garbage collection

    pj, err := ctx.Create(`
        +proj=pipeline
        +step +proj=unitconvert +xy_in=deg +xy_out=rad
        +step +proj=utm +zone=31
    `)
    if err != nil {
        log.Fatal(err)
    }
    defer pj.Close() // if omitted, this will be called on garbage collection

    x, y, _, _, err := pj.Trans(proj.Fwd, 3, 58, 0, 0)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(x, y)

*/
package proj
