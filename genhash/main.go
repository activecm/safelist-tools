package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/globalsign/mgo/bson"
)

type (

	// Entry defines an entry in the global safelist. The entries must be serializable
	// to both JSON and BSON for exporting the safelist to a json file and writing the safelist to MongoDB.
	// The type field defines which of the possible entry definitions exists within the structure.
	Entry struct {
		ObjectID bson.ObjectId `bson:"_id,omitempty" json:"_id,omitempty"`

		Name string `bson:"name" json:"name"`

		Type string `bson:"type" json:"type"`

		HashKey int64 `bson:"hash_key" json:"hash_key"` // 64 bit hash of entry's identifying information

		Comment string `bson:"comment" json:"comment"`

		SchemaVersion int `bson:"schema_version" json:"schema_version"`

		IP *IPEntry `bson:"ip,omitempty" json:"ip,omitempty"`

		IPPair *IPPairEntry `bson:"pair,omitempty" json:"pair,omitempty"`

		IPRanges *IPRangesEntry `bson:"ranges,omitempty" json:"ranges,omitempty"`

		Domain string `bson:"domain,omitempty" json:"domain,omitempty"`

		Useragent string `bson:"useragent,omitempty" json:"useragent,omitempty"`
	}

	EntryIPRange struct {
		Start uint32 `bson:"start" json:"start"`
		End   uint32 `bson:"end" json:"end"`
	}

	IPEntry struct {
		IP        string      `bson:"ip" json:"ip"`
		NetworkID bson.Binary `bson:"network_uuid" json:"network_uuid"`
		Src       bool        `bson:"src" json:"src"`
		Dst       bool        `bson:"dst" json:"dst"`
	}

	IPPairEntry struct {
		SrcIP          string      `bson:"src" json:"src"`
		SrcNetworkUUID bson.Binary `bson:"src_network_uuid" json:"src_network_uuid"`
		DstIP          string      `bson:"dst" json:"dst"`
		DstNetworkUUID bson.Binary `bson:"dst_network_uuid" json:"dst_network_uuid"`
	}

	IPRangesEntry struct {
		Ranges    []EntryIPRange `bson:"ranges" json:"ranges"`
		NetworkID bson.Binary    `bson:"network_uuid" json:"network_uuid"`
		Src       bool           `bson:"src" json:"src"`
		Dst       bool           `bson:"dst" json:"dst"`
	}
)

//HashKey returns an int64 which uniquely identfies this IPEntry
func (i *IPEntry) HashKey() (int64, error) {
	unsignedHash := fnv.New64a()
	_, err := unsignedHash.Write([]byte(i.IP))

	if err != nil {
		return 0, err
	}
	_, err = unsignedHash.Write(i.NetworkID.Data)

	if err != nil {
		return 0, err
	}

	return int64(unsignedHash.Sum64()), nil
}

//HashKey returns an int64 which uniquely identfies this IPPairEntry
func (i *IPPairEntry) HashKey() (int64, error) {
	unsignedHash := fnv.New64a()
	_, err := unsignedHash.Write([]byte(i.SrcIP))
	if err != nil {
		return 0, err
	}

	_, err = unsignedHash.Write(i.SrcNetworkUUID.Data)
	if err != nil {
		return 0, err
	}

	_, err = unsignedHash.Write([]byte(i.DstIP))
	if err != nil {
		return 0, err
	}

	_, err = unsignedHash.Write(i.DstNetworkUUID.Data)
	if err != nil {
		return 0, err
	}
	return int64(unsignedHash.Sum64()), nil
}

//HashKey returns an int64 which uniquely identfies this IPRangesEntry
func (i *IPRangesEntry) HashKey() (int64, error) {
	// unordered hash for the ranges array
	var rangeHash uint64
	for _, rng := range i.Ranges {
		innerHasher := fnv.New64a()
		buf := make([]byte, 8)
		binary.BigEndian.PutUint32(buf[0:4], rng.Start)
		binary.BigEndian.PutUint32(buf[4:8], rng.End)

		_, err := innerHasher.Write(buf)
		if err != nil {
			return 0, err
		}

		rangeHash += innerHasher.Sum64()
	}

	unsignedHash := fnv.New64a()

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, rangeHash)

	_, err := unsignedHash.Write(buf)
	if err != nil {
		return 0, err
	}

	_, err = unsignedHash.Write(i.NetworkID.Data)
	if err != nil {
		return 0, err
	}
	return int64(unsignedHash.Sum64()), nil
}

//StringHashKey returns an int64 which uniquely identfies this UseragentEntry or DomainEntry
func StringHashKey(i string) (int64, error) {
	unsignedHash := fnv.New64a()
	_, err := unsignedHash.Write([]byte(i))

	if err != nil {
		return 0, err
	}

	return int64(unsignedHash.Sum64()), nil
}

// loadSafelist reads in a file that contains
// an array of JSON entries
func loadSafelist(file string) ([]Entry, error) {
	jsonFile, fileErr := os.Open(file)

	if fileErr != nil {
		return nil, fileErr
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	// Unmarshal the JSON into a list of safelist.Entry structs
	var safelistDocument []Entry
	jsonErr := json.Unmarshal(byteValue, &safelistDocument)

	if jsonErr != nil {
		return nil, jsonErr
	}

	return safelistDocument, nil
}

// processSafelist will look for safelist entries
// that do no have a HashKey present, generate the
// HashKey for those entries, and update the safelistDocument
// array with the new HashKey values
func processSafelist(safelistDocument []Entry) {

	for idx, currEntry := range safelistDocument {
		// For each entry, determine the type and
		// calculate the hash for that entry type.
		// I know that breaks aren't needed everywhere,
		// just put them in to make things clearer with the
		// fallthrough cases added
		var err error
		if currEntry.HashKey == 0 {
			if currEntry.SchemaVersion == 0 {
				fmt.Println("[*] SchemaVersion missing in this entry: ", currEntry)
				fmt.Println("[+] Setting the SchemaVersion to 5", currEntry)
				safelistDocument[idx].SchemaVersion = 5
			}
			switch entryType := strings.ToLower(currEntry.Type); entryType {

			// Range type entries
			case "asn":
				fallthrough
			case "asn_org":
				fallthrough
			case "cidr":
				fallthrough
			case "ranges":

				if currEntry.IPRanges.NetworkID.Kind == 0 || currEntry.IPRanges.NetworkID.Data == nil || currEntry.IPRanges.Ranges == nil {
					fmt.Println("[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = currEntry.IPRanges.HashKey()
				break

			// Domain type entries
			case "domain_literal":
				fallthrough
			case "domain_pattern":

				if currEntry.Domain == "" {
					fmt.Println("[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = StringHashKey(currEntry.Domain)
				break

			// Single IP entry
			case "ip":

				if currEntry.IP.IP == "" || currEntry.IP.NetworkID.Data == nil || currEntry.IP.NetworkID.Kind == 0 {
					fmt.Println("[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = currEntry.IP.HashKey()
				break

			// IP pair entry
			case "pair":

				if currEntry.IPPair.DstIP == "" || currEntry.IPPair.SrcIP == "" ||
					currEntry.IPPair.DstNetworkUUID.Data == nil || currEntry.IPPair.DstNetworkUUID.Kind == 0 ||
					currEntry.IPPair.SrcNetworkUUID.Data == nil || currEntry.IPPair.SrcNetworkUUID.Kind == 0 {
					fmt.Println("[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = currEntry.IPPair.HashKey()
				break

			// Useragent entry
			case "useragent":
				if currEntry.Useragent == "" {
					fmt.Println("[*] Missing information in this entry, skipping:", currEntry)
					continue
				}
				safelistDocument[idx].HashKey, err = StringHashKey(currEntry.Useragent)
				break

			}

			if err != nil {
				fmt.Println("[*] Error hashing this entry:", currEntry)
				fmt.Println("[*] Error message generated from hasher:", err)
			}

		}
	}
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("[-] Usage: ./genhash inputFilename [outputFilename]")
		os.Exit(-1)
	}

	inputFilename := os.Args[1]

	outputFilename := strings.TrimSuffix(inputFilename, filepath.Ext(inputFilename)) + "-hashed.json"

	if len(os.Args) == 3 {
		outputFilename = os.Args[2]
	}

	safelistDocument, fileReadErr := loadSafelist(inputFilename)

	if fileReadErr != nil {
		fmt.Printf("[*] Error reading data from %s: %s", inputFilename, fileReadErr)
	}

	processSafelist(safelistDocument[:])

	jsonData, _ := json.Marshal(safelistDocument)

	fileWriteErr := os.WriteFile(outputFilename, jsonData, 0644)

	if fileWriteErr != nil {
		fmt.Printf("[*] Error saving to file %s: %s", outputFilename, fileWriteErr)
	}
}
