CREATE TABLE IF NOT EXISTS emg_sensor_data (
    user_id UInt32,
    prosthesis_type String,
    muscle_group String,
    signal_frequency UInt32,
    signal_duration UInt32,
    signal_amplitude Decimal(5,2),
    signal_time DateTime
) ENGINE = MergeTree()
ORDER BY (user_id, prosthesis_type, signal_time);

INSERT INTO emg_sensor_data
SELECT *
FROM file('olap.csv', 'CSV');


CREATE DATABASE IF NOT EXISTS reports;

CREATE TABLE IF NOT EXISTS reports.crm_customers
(
    id UInt32,
    name String,
    email String,
    age UInt16,
    gender LowCardinality(String),
    country LowCardinality(String),
    address String,
    phone String
)
ENGINE = MergeTree()
ORDER BY id;

CREATE TABLE IF NOT EXISTS reports.user_emg_daily
(
    day Date,
    user_id UInt32,
    prosthesis_type LowCardinality(String),
    muscle_group LowCardinality(String),
    signals UInt64,
    avg_amplitude Float64,
    p95_amplitude Float64,
    avg_frequency Float64,
    avg_duration Float64,
    first_signal DateTime,
    last_signal DateTime
)
ENGINE = MergeTree()
ORDER BY (user_id, day, prosthesis_type, muscle_group);

CREATE TABLE IF NOT EXISTS reports.user_report_mart
(
    day Date,
    user_id UInt32,
    name String,
    email String,
    age UInt16,
    gender LowCardinality(String),
    country LowCardinality(String),
    prosthesis_type LowCardinality(String),
    muscle_group LowCardinality(String),
    signals UInt64,
    avg_amplitude Float64,
    p95_amplitude Float64,
    avg_frequency Float64,
    avg_duration Float64,
    first_signal DateTime,
    last_signal DateTime
)
ENGINE = MergeTree()
ORDER BY (user_id, day, prosthesis_type, muscle_group);

CREATE MATERIALIZED VIEW IF NOT EXISTS reports.mv_user_emg_daily
TO reports.user_emg_daily AS
SELECT
    toDate(signal_time) AS day,
    user_id,
    prosthesis_type,
    muscle_group,
    count() AS signals,
    avg(toFloat64(signal_amplitude)) AS avg_amplitude,
    quantileTDigest(0.95)(toFloat64(signal_amplitude)) AS p95_amplitude,
    avg(toFloat64(signal_frequency)) AS avg_frequency,
    avg(toFloat64(signal_duration)) AS avg_duration,
    min(signal_time) AS first_signal,
    max(signal_time) AS last_signal
FROM default.emg_sensor_data
GROUP BY day, user_id, prosthesis_type, muscle_group;



INSERT INTO reports.user_report_mart
SELECT *
FROM file('user_report_mart_sample.csv', 'CSV');