
from datetime import datetime, timedelta
from airflow import DAG
from airflow.operators.python import PythonOperator
from airflow.hooks.base import BaseHook
from airflow.providers.postgres.hooks.postgres import PostgresHook
import logging

# pip install clickhouse-driver psycopg2-binary
try:
    from clickhouse_driver import Client as CHClient
except Exception as e:
    CHClient = None
    logging.warning("clickhouse-driver not available: %s", e)

CRM_CONN_ID = "crm_postgres"
CH_CONN_ID = "olap_clickhouse"

default_args = {
    "owner": "data-eng",
    "retries": 1,
    "retry_delay": timedelta(minutes=5),
}

with DAG(
    dag_id="reports_etl",
    default_args=default_args,
    start_date=datetime(2025, 9, 1),
    schedule_interval="0 * * * *",  # Run every hour
    catchup=False,
    tags=["bionicpro", "etl", "reports"],
) as dag:

    def _get_ch_client():
        """Get ClickHouse client connection"""
        if CHClient is None:
            raise RuntimeError("clickhouse-driver is not installed")
        
        # Use direct connection parameters for ClickHouse
        return CHClient(
            host='olap_db',
            port=9000,
            user='default',
            password='',
            database='reports'
        )

    def extract_crm_to_clickhouse(**context):
        """Extract data from CRM PostgreSQL and load into ClickHouse"""
        if CHClient is None:
            raise RuntimeError("clickhouse-driver is not installed")
        
        # Connect to CRM PostgreSQL
        pg = PostgresHook(postgres_conn_id=CRM_CONN_ID)
        
        # Get CRM data
        query = """
        SELECT id, name, email, 
               COALESCE(age::int, 0) as age, 
               COALESCE(gender, '') as gender, 
               COALESCE(country, '') as country, 
               COALESCE(address, '') as address, 
               COALESCE(phone, '') as phone 
        FROM customers
        """
        
        rows = pg.get_records(query)
        logging.info("Fetched %d CRM rows", len(rows))

        # Connect to ClickHouse
        ch = _get_ch_client()
        
        # Clear existing CRM data
        ch.execute("TRUNCATE TABLE IF EXISTS reports.crm_customers")
        
        # Insert new data
        if rows:
            ch.execute(
                "INSERT INTO reports.crm_customers (id, name, email, age, gender, country, address, phone) VALUES",
                rows
            )
            logging.info("Inserted %d rows into reports.crm_customers", len(rows))

    def build_user_report_mart(**context):
        """Build the user report mart by joining CRM and EMG data"""
        if CHClient is None:
            raise RuntimeError("clickhouse-driver is not installed")
        
        ch = _get_ch_client()
        
        # Delete recent data to avoid duplicates
        delete_sql = "ALTER TABLE reports.user_report_mart DELETE WHERE day >= today() - 1;"
        ch.execute(delete_sql)
        logging.info("Deleted recent data from user_report_mart")
        
        # Insert new data by joining CRM customers with EMG daily data
        insert_sql = """
        INSERT INTO reports.user_report_mart
        SELECT
            d.day,
            d.user_id,
            c.name,
            c.email,
            c.age,
            c.gender,
            c.country,
            d.prosthesis_type,
            d.muscle_group,
            d.signals,
            d.avg_amplitude,
            d.p95_amplitude,
            d.avg_frequency,
            d.avg_duration,
            d.first_signal,
            d.last_signal
        FROM reports.user_emg_daily AS d
        LEFT JOIN reports.crm_customers AS c ON d.user_id = c.id
        WHERE d.day >= today() - 1
        """
        
        ch.execute(insert_sql)
        logging.info("Built user_report_mart with joined data")

    def check_data_quality(**context):
        """Check data quality and log statistics"""
        if CHClient is None:
            raise RuntimeError("clickhouse-driver is not installed")
        
        ch = _get_ch_client()
        
        # Get counts
        crm_count = ch.execute("SELECT COUNT(*) FROM reports.crm_customers")[0][0]
        emg_count = ch.execute("SELECT COUNT(*) FROM reports.user_emg_daily")[0][0]
        mart_count = ch.execute("SELECT COUNT(*) FROM reports.user_report_mart")[0][0]
        
        logging.info(f"Data quality check - CRM: {crm_count}, EMG: {emg_count}, Mart: {mart_count}")
        
        if mart_count == 0:
            raise ValueError("No data in user_report_mart after ETL process")

    # Define tasks
    extract_crm = PythonOperator(
        task_id="extract_crm_to_clickhouse", 
        python_callable=extract_crm_to_clickhouse
    )
    
    build_mart = PythonOperator(
        task_id="build_user_report_mart", 
        python_callable=build_user_report_mart
    )
    
    data_quality_check = PythonOperator(
        task_id="check_data_quality", 
        python_callable=check_data_quality
    )

    # Define task dependencies
    extract_crm >> build_mart >> data_quality_check
