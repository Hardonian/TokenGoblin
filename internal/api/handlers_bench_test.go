package api

import (
	"fmt"
	"strings"
	"testing"
	"strconv"
)

func BenchmarkReportStringFormatting(b *testing.B) {
	type Driver struct {
		Type         string
		Key          string
		TotalCostUSD float64
		EventCount   int64
	}
	type Issue struct {
		Code    string
		Message string
	}
	type Analysis struct {
		EventID string
		Issues  []Issue
	}

	summary := struct {
		TopCostDrivers []Driver
	}{
		TopCostDrivers: []Driver{
			{"model", "gpt-4", 12.3456, 100},
			{"model", "gpt-3.5-turbo", 1.2345, 500},
		},
	}
	analyses := []Analysis{
		{
			EventID: "evt_123",
			Issues: []Issue{
				{"high_latency", "Took 5 seconds to process"},
				{"high_cost", "Cost $1.00 for a single request"},
			},
		},
		{
			EventID: "evt_456",
			Issues: []Issue{
				{"error", "Request failed"},
			},
		},
	}

	b.Run("Sprintf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var sb strings.Builder
			for _, driver := range summary.TopCostDrivers {
				sb.WriteString(fmt.Sprintf("- %s `%s`: estimated `$%.4f` across `%d` events\n", driver.Type, driver.Key, driver.TotalCostUSD, driver.EventCount))
			}
			for _, item := range analyses {
				for _, issue := range item.Issues {
					sb.WriteString(fmt.Sprintf("- `%s` on event `%s`: %s\n", issue.Code, item.EventID, issue.Message))
				}
			}
		}
	})

	b.Run("WriteString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var sb strings.Builder
			for _, driver := range summary.TopCostDrivers {
				sb.WriteString("- ")
				sb.WriteString(driver.Type)
				sb.WriteString(" `")
				sb.WriteString(driver.Key)
				sb.WriteString("`: estimated `$")
				sb.WriteString(strconv.FormatFloat(driver.TotalCostUSD, 'f', 4, 64))
				sb.WriteString("` across `")
				sb.WriteString(strconv.FormatInt(driver.EventCount, 10))
				sb.WriteString("` events\n")
			}
			for _, item := range analyses {
				for _, issue := range item.Issues {
					sb.WriteString("- `")
					sb.WriteString(issue.Code)
					sb.WriteString("` on event `")
					sb.WriteString(item.EventID)
					sb.WriteString("`: ")
					sb.WriteString(issue.Message)
					sb.WriteString("\n")
				}
			}
		}
	})
}
