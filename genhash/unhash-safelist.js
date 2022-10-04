const originalSafelist = require("./safelist-orig.json")
const fs = require("fs")
const { exit } = require("process")

function unhash() {
    // delete the unhashed safelist for convenience when testing locally
    fs.unlinkSync("./safelist-unhashed.json")

    // create a new copy of the original safelist and remove the hash_key property
    const unhashedSafelist = originalSafelist.map(({hash_key, ...rest}) => rest)
    console.log(unhashedSafelist.length == originalSafelist.length)

    // write unhashed safelist to json file
    const data = JSON.stringify(unhashedSafelist)
    fs.writeFileSync("safelist-unhashed.json", data, {flag: "wx"}, function(err) {
        if(err) {
            console.log(err)
            exit(-1)
        }
    })
}

unhash()