package krakow

import (
	"fmt"
	"os"
	"strings"
	"time"
	wheretopark "wheretopark/go"

	"github.com/rs/zerolog/log"
	"github.com/xuri/excelize/v2"
)

type Solari struct {
	basePath string
}

func NewSolari(basePath string) Source {
	return Solari{
		basePath: basePath,
	}
}

func (s Solari) Files() ([]string, error) {
	return ListFilesByExtension(s.basePath, ".xlsx")
}

func (f Solari) LoadFile(file *os.File) (map[meterCode][]meterRecord, error) {
	spreadsheet, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("error reading spreadsheet: %w", err)
	}
	defer spreadsheet.Close()

	rows, err := spreadsheet.GetRows("Transactions")
	if err != nil {
		return nil, err
	}

	records := make(map[meterCode][]meterRecord, len(rows))
	for i, row := range rows {
		if i == 0 {
			continue
		}
		code := row[1]
		zone := row[2]
		subzone := row[3]
		dateStr := row[4]
		durationStr := row[5]
		if durationStr == "" || !strings.Contains(dateStr, "/") {
			log.Debug().Str("row", fmt.Sprintf("%v", row)).Msg("skipping entry")
			continue
		}
		startDate := wheretopark.MustParseDateTimeWith(solariDateFormat, dateStr)
		duration := wheretopark.Must(parseSolariDuration(durationStr))

		if _, ok := records[code]; !ok {
			records[code] = make([]meterRecord, 0, 8)
		}
		records[code] = append(records[code], meterRecord{
			zone:      zone,
			subzone:   subzone,
			startDate: startDate,
			endDate:   startDate.Add(duration),
		})
	}

	return records, nil
}

const solariDateFormat = "1/2/06 15:04"

func parseSolariDuration(str string) (time.Duration, error) {
	a := strings.Split(str, ":")
	h := a[0]
	m := a[1]
	s := a[2]
	if s != "00" {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	strDuration := fmt.Sprintf("%sh%sm", h, m)
	return time.ParseDuration(strDuration)
}
