package aws

import (
	"fmt"
	"sort"
	"strings"

	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type recordSorter []route53Types.ResourceRecordSet

func (rs recordSorter) Len() int      { return len(rs) }
func (rs recordSorter) Swap(i, j int) { rs[i], rs[j] = rs[j], rs[i] }
func (rs recordSorter) Less(i, j int) bool {
	if *rs[i].Name != *rs[j].Name {
		return *rs[i].Name < *rs[j].Name
	}
	return string(rs[i].Type) < string(rs[j].Type)
}

func GenerateZoneFile(records []route53Types.ResourceRecordSet, zoneName string) (string, error) {
	if len(records) == 0 {
		return "", nil
	}

	var sb strings.Builder
	var soaRecord *route53Types.ResourceRecordSet
	var apexNsRecords []route53Types.ResourceRecordSet
	var otherRecords []route53Types.ResourceRecordSet

	if !strings.HasSuffix(zoneName, ".") {
		zoneName += "."
	}

	for _, rec := range records {
		if rec.AliasTarget != nil {
			sb.WriteString(fmt.Sprintf("; ALIAS record skipped (not standard): %s -> %s\n", *rec.Name, *rec.AliasTarget.DNSName))
			continue
		}

		recordType := rec.Type
		recordName := *rec.Name

		if recordType == "SOA" && recordName == zoneName {
			soaRecord = &rec
		} else if recordType == "NS" && recordName == zoneName {
			apexNsRecords = append(apexNsRecords, rec)
		} else {
			otherRecords = append(otherRecords, rec)
		}
	}

	if soaRecord == nil {
		return "", fmt.Errorf("SOA record for zone %s not found", zoneName)
	}

	soaValue := *soaRecord.ResourceRecords[0].Value
	soaParts := strings.Fields(soaValue)
	if len(soaParts) < 7 {
		return "", fmt.Errorf("invalid SOA record value: %s", soaValue)
	}
	defaultTTL := soaParts[6]
	sb.WriteString(fmt.Sprintf("$TTL %s\n", defaultTTL))

	sb.WriteString(fmt.Sprintf("@\t%d\tIN\tSOA\t%s\n\n", *soaRecord.TTL, soaValue))

	for _, ns := range apexNsRecords {
		for _, val := range ns.ResourceRecords {
			sb.WriteString(fmt.Sprintf("@\t%d\tIN\tNS\t%s\n", *ns.TTL, *val.Value))
		}
	}
	sb.WriteString("\n")

	sort.Sort(recordSorter(otherRecords))

	for _, rec := range otherRecords {
		for _, val := range rec.ResourceRecords {
			formattedValue, err := formatRecordValue(string(rec.Type), *val.Value)
			if err != nil {
				return "", err
			}

			recordName := *rec.Name
			if recordName == zoneName {
				recordName = "@"
			}

			sb.WriteString(fmt.Sprintf(
				"%s\t%d\tIN\t%s\t%s\n",
				recordName,
				*rec.TTL,
				rec.Type,
				formattedValue,
			))
		}
	}

	return sb.String(), nil
}

func formatRecordValue(recordType, value string) (string, error) {
	switch recordType {
	case "TXT":
		//return strconv.Quote(value), nil
		return value, nil
	default:
		return value, nil
	}
}
