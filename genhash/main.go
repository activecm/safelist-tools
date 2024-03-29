package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
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

		IPPair       *IPPairEntry       `bson:"pair,omitempty" json:"pair,omitempty"`
		IPPairRanges *IPPairRangesEntry `bson:"pair_ranges,omitempty" json:"pair_ranges,omitempty"`

		IPRanges *IPRangesEntry `bson:"ranges,omitempty" json:"ranges,omitempty"`

		Domain           string                 `bson:"domain,omitempty" json:"domain,omitempty"`
		DomainPair       *DomainPairEntry       `bson:"domain_pair,omitempty" json:"domain_pair,omitempty"`
		DomainPairRanges *DomainPairRangesEntry `bson:"domain_pair_ranges,omitempty" json:"domain_pair_ranges,omitempty"`

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

	IPPairRangesEntry struct {
		SrcRanges      []EntryIPRange `bson:"src_ranges" json:"src_ranges"`
		SrcNetworkUUID bson.Binary    `bson:"src_network_uuid" json:"src_network_uuid"`
		DstRanges      []EntryIPRange `bson:"dst_ranges" json:"dst_ranges"`
		DstNetworkUUID bson.Binary    `bson:"dst_network_uuid" json:"dst_network_uuid"`
	}

	IPRangesEntry struct {
		Ranges    []EntryIPRange `bson:"ranges" json:"ranges"`
		NetworkID bson.Binary    `bson:"network_uuid" json:"network_uuid"`
		Src       bool           `bson:"src" json:"src"`
		Dst       bool           `bson:"dst" json:"dst"`
	}

	DomainPairSrcEntry struct {
		IP        string      `bson:"ip" json:"ip"`
		NetworkID bson.Binary `bson:"network_uuid" json:"network_uuid"`
	}

	DomainPairEntry struct {
		Src  *DomainPairSrcEntry `bson:"src" json:"src"`
		FQDN string              `bson:"fqdn" json:"fqdn"`
	}

	EntryDomainPairRange struct {
		Start uint32 `bson:"start" json:"start"`
		End   uint32 `bson:"end" json:"end"`
	}

	DomainPairRangesEntry struct {
		NetworkID bson.Binary    `bson:"network_uuid" json:"network_uuid"`
		FQDN      string         `bson:"fqdn" json:"fqdn"`
		Ranges    []EntryIPRange `bson:"ranges" json:"ranges"`
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

func (i *IPPairRangesEntry) HashKey() (int64, error) {
	// unordered hash for the ranges array
	var rangeHash uint64

	hashes := append(i.SrcRanges, i.DstRanges...)

	for _, rng := range hashes {
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

	_, err = unsignedHash.Write(i.SrcNetworkUUID.Data)
	if err != nil {
		return 0, err
	}

	_, err = unsignedHash.Write(i.DstNetworkUUID.Data)
	if err != nil {
		return 0, err
	}

	return int64(unsignedHash.Sum64()), nil
}

func (i *DomainPairEntry) HashKey() (int64, error) {
	unsignedHash := fnv.New64a()
	_, err := unsignedHash.Write([]byte(i.FQDN))
	if err != nil {
		return 0, err
	}

	_, err = unsignedHash.Write([]byte(i.Src.IP))
	if err != nil {
		return 0, err
	}

	_, err = unsignedHash.Write(i.Src.NetworkID.Data)
	if err != nil {
		return 0, err
	}

	return int64(unsignedHash.Sum64()), nil
}

func (i *DomainPairRangesEntry) HashKey() (int64, error) {
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

	_, err = unsignedHash.Write([]byte(i.FQDN))
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
				fmt.Fprintln(os.Stderr, "[*] SchemaVersion missing in this entry: ", currEntry)
				fmt.Fprintln(os.Stderr, "[+] Setting the SchemaVersion to 5", currEntry)
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
					fmt.Fprintln(os.Stderr, "[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = currEntry.IPRanges.HashKey()
				break

			// Domain type entries
			case "domain_literal":
				fallthrough
			case "domain_pattern":

				if currEntry.Domain == "" {
					fmt.Fprintln(os.Stderr, "[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = StringHashKey(currEntry.Domain)
				break
			// IP -> Domain pair type entries
			case "domain_pair_literal":
				fallthrough
			case "domain_pair_pattern":

				if currEntry.DomainPair.Src.IP == "" || currEntry.DomainPair.Src.NetworkID.Data == nil || currEntry.DomainPair.Src.NetworkID.Kind == 0 ||
					currEntry.DomainPair.FQDN == "" {
					fmt.Fprintln(os.Stderr, "[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = currEntry.DomainPair.HashKey()
				break

			// CIDR -> Domain pair type entries
			case "domain_pair_cidr_literal":
				fallthrough
			case "domain_pair_cidr_pattern":
				fallthrough
			case "domain_pair_ranges_literal":
				fallthrough
			case "domain_pair_ranges_pattern":
				if currEntry.DomainPairRanges.NetworkID.Data == nil || currEntry.DomainPairRanges.NetworkID.Kind == 0 || currEntry.DomainPairRanges.FQDN == "" ||
					currEntry.DomainPairRanges.Ranges == nil {
					fmt.Fprintln(os.Stderr, "[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = currEntry.DomainPairRanges.HashKey()
				break

			// Single IP entry
			case "ip":

				if currEntry.IP.IP == "" || currEntry.IP.NetworkID.Data == nil || currEntry.IP.NetworkID.Kind == 0 {
					fmt.Fprintln(os.Stderr, "[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = currEntry.IP.HashKey()
				break

			// IP pair entry
			case "pair":

				if currEntry.IPPair.DstIP == "" || currEntry.IPPair.SrcIP == "" ||
					currEntry.IPPair.DstNetworkUUID.Data == nil || currEntry.IPPair.DstNetworkUUID.Kind == 0 ||
					currEntry.IPPair.SrcNetworkUUID.Data == nil || currEntry.IPPair.SrcNetworkUUID.Kind == 0 {
					fmt.Fprintln(os.Stderr, "[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = currEntry.IPPair.HashKey()
				break

			// CIDR pair entry
			case "pair_cidr":
				fallthrough
			case "pair_ranges":
				if currEntry.IPPairRanges.SrcNetworkUUID.Kind == 0 || currEntry.IPPairRanges.SrcNetworkUUID.Data == nil ||
					currEntry.IPPairRanges.DstNetworkUUID.Kind == 0 || currEntry.IPPairRanges.DstNetworkUUID.Data == nil ||
					currEntry.IPPairRanges.SrcRanges == nil || currEntry.IPPairRanges.DstRanges == nil {
					fmt.Fprintln(os.Stderr, "[*] Missing information in this entry, skipping:", currEntry)
					continue
				}

				safelistDocument[idx].HashKey, err = currEntry.IPPairRanges.HashKey()
				break

			// Useragent entry
			case "useragent":
				if currEntry.Useragent == "" {
					fmt.Fprintln(os.Stderr, "[*] Missing information in this entry, skipping:", currEntry)
					continue
				}
				safelistDocument[idx].HashKey, err = StringHashKey(currEntry.Useragent)
				break

			}

			if err != nil {
				fmt.Fprintln(os.Stderr, "[*] Error hashing this entry:", currEntry)
				fmt.Fprintln(os.Stderr, "[*] Error message generated from hasher:", err)
			}

		}
	}
}

func main() {

	var safelistDocument []Entry

	// Check for CI env variable flag, set by Github Actions
	isCI, err := strconv.ParseBool(os.Getenv("CI"))
	if err != nil {
		isCI = false
	}
	// Handle reading data from pipe via standard in
	inPipe, err := os.Stdin.Stat()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input")
	}

	// Don't pipe in via stdin if running in Github Actions
	if (inPipe.Mode()&os.ModeCharDevice == 0) && !isCI {
		reader := bufio.NewReader(os.Stdin)
		byteValue := make([]byte, 0, 16384)
		currByte, err := reader.ReadByte()

		for err != io.EOF {
			byteValue = append(byteValue, currByte)
			currByte, err = reader.ReadByte()
		}

		json.Unmarshal(byteValue, &safelistDocument)

		processSafelist(safelistDocument[:])

		jsonData, _ := json.Marshal(safelistDocument)

		fmt.Print(string(jsonData[:]))

	} else {

		// Handle if arguments were passed in instead
		if len(os.Args) < 2 {
			fmt.Fprintln(os.Stderr, "[-] Usage: ./genhash inputFilename [outputFilename]")
			os.Exit(-1)
		}

		inputFilename := os.Args[1]

		outputFilename := strings.TrimSuffix(inputFilename, filepath.Ext(inputFilename)) + "-hashed.json"

		if len(os.Args) == 3 {
			outputFilename = os.Args[2]
		}

		safelistDocument, fileReadErr := loadSafelist(inputFilename)

		if fileReadErr != nil {
			fmt.Fprintf(os.Stderr, "[*] Error reading data from %s: %s\n", inputFilename, fileReadErr)
		}
		processSafelist(safelistDocument[:])

		jsonData, _ := json.Marshal(safelistDocument)

		fileWriteErr := os.WriteFile(outputFilename, jsonData, 0644)

		if fileWriteErr != nil {
			fmt.Fprintf(os.Stderr, "[*] Error saving to file %s: %s\n", outputFilename, fileWriteErr)
		}
	}
}
