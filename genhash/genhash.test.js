const originalSafelist = require('./safelist-orig.json')
const unhashedSafelist = require('./safelist-unhashed.json')
const generatedSafelist = require('./safelist-generated.json')

// beforeAll(() => {
//     const build = new Promise((resolve, reject) => {
//         const make = spawn("make")
//         make.on("error", (err) => {
//             reject()
//         })
//         make.on("close", code => {
//             if(code == 0) {
//                 resolve()
//             }
//             reject()
//         })
//     }) 
    
//     const generateHashes = new Promise((resolve, reject) => {
//         const genhash = spawn("./genhash", ["safelist-unhashed.json", "generated-safelist.json"])
//         genhash.on("close", code => {
//             if(code == 0) {
//                 console.log("done")
//                 resolve()
//             } else {
//                 console.error("hash generation failed")
//                 reject()
//             }
//         })
//     })

//     function setup() {

//     }
//    return build.then(() => generateHashes)
        
// })
    
test('unhashed safelist is valid', () => {
    expect(unhashedSafelist.length).toBe(originalSafelist.length)
    originalSafelist.forEach((entry) => {
        // find matching safelist entry in the generated safelist
        const matchingEntry = unhashedSafelist.find((e) => e.type == entry.type && e.name == entry.name)
        expect(matchingEntry).not.toBeUndefined()
    })
})

test('generates correct hashes', () => {
    // make sure the original safelist and the generated safelist are the same length
    expect(generatedSafelist.length).toBe(originalSafelist.length)

    // compare each safelist entry from the original safelist and ensure there's a matching
    // safelist entry and hash in the generated safelist
    originalSafelist.forEach((entry) => {
        // find matching safelist entry in the generated safelist
        const matchingEntry = generatedSafelist.find((e) => e.type == entry.type && e.name == entry.name)
        expect(matchingEntry).not.toBeUndefined()

        // check if the hashes match
        const generatedHash = matchingEntry.hash_key.toString()
        const originalHash = entry.hash_key.toString()
        expect(generatedHash).toEqual(originalHash)
    })
})