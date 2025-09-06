package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickHouseClient struct {
	conn driver.Conn
}

type ReportRecord struct {
	ProsthesisType string    `json:"prosthesis_type"`
	MuscleGroup    string    `json:"muscle_group"`
	Signals        uint64    `json:"signals"` // Changed from int64 to uint64
	AvgAmplitude   float64   `json:"avg_amplitude"`
	P95Amplitude   float64   `json:"p95_amplitude"`
	AvgFrequency   float64   `json:"avg_frequency"`
	AvgDuration    float64   `json:"avg_duration"`
	FirstSignal    time.Time `json:"first_signal"`
	LastSignal     time.Time `json:"last_signal"`
}

func NewClickHouseClient(config *Config) (*ClickHouseClient, error) {
	conn, _ := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", config.ClickHouseHost, config.ClickHousePort)},
		Auth: clickhouse.Auth{
			Database: config.ClickHouseDB,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
	})

	/*if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}*/

	return &ClickHouseClient{conn: conn}, nil
}

func (c *ClickHouseClient) GetUserReports(ctx context.Context, userID string) ([]ReportRecord, error) {
	log.Println("userID", userID)
	query := `
		SELECT prosthesis_type,muscle_group,signals,avg_amplitude,p95_amplitude,avg_frequency,avg_duration,first_signal,last_signal
		FROM reports.user_report_mart
		WHERE email = ?
		ORDER BY day DESC
		LIMIT 1000
	`

	rows, err := c.conn.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var reports []ReportRecord
	for rows.Next() {
		log.Println("123")
		var record ReportRecord
		if err := rows.Scan(
			&record.ProsthesisType,
			&record.MuscleGroup,
			&record.Signals,
			&record.AvgAmplitude,
			&record.P95Amplitude,
			&record.AvgFrequency,
			&record.AvgDuration,
			&record.FirstSignal,
			&record.LastSignal,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		reports = append(reports, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return reports, nil
}

func (c *ClickHouseClient) Close() error {
	return c.conn.Close()
}
