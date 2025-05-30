import pandas as pd
import psycopg2
import httpx
from datetime import datetime, timedelta
from dotenv import load_dotenv
import os

load_dotenv()

def main():
    # Simple raw SQL - just get the data we need
    query = """
    SELECT 
        DATE(p.created_at) as date,
        u.email as paid_to_email,
        u.payment_address as paid_to_address,
        p.price as price_usdc,
        p.is_test
    FROM purchases p
    JOIN paid_routes pr ON p.paid_route_id = pr.id
    JOIN users u ON pr.user_id = u.id
    WHERE p.created_at >= CURRENT_DATE - INTERVAL '8 days'
    """
    
    try:
        conn = psycopg2.connect(
            host=os.getenv("DB_HOST", "localhost"),
            port=os.getenv("DB_PORT", "5432"), 
            user=os.getenv("DB_USER"),
            password=os.getenv("DB_PASSWORD"),
            dbname=os.getenv("DB_NAME")
        )
        df = pd.read_sql(query, conn)
        conn.close()
    except Exception as e:
        send_error(f"Database error: {e}")
        return
    
    # Filter and group the data
    today = datetime.now().date()
    week_start = today - timedelta(days=6)
    
    grouped_df = df[df['date'] >= week_start].groupby(['date', 'is_test']).agg(
        emails=('paid_to_email', lambda x: list(set(x))),
        total_usdc=('price_usdc', lambda x: x.sum() / 1_000_000),
        total_txs=('price_usdc', 'count')
    ).reset_index().sort_values(['date', 'is_test'], ascending=[False, False])
    
    # Format as table for Slack
    table_text = f"```\n{grouped_df.to_string(index=False)}\n```"
    message = f"📈 Daily Payment Stats\n\n{table_text}"
    
    # Send to Slack
    webhook_url = os.getenv("SLACK_WEBHOOK_URL")
    if webhook_url:
        try:
            httpx.post(webhook_url, json={"text": message}, timeout=10)
            print("Report sent to Slack successfully")
        except Exception as e:
            print(f"Failed to send to Slack: {e}")
    else:
        print("No Slack webhook configured - printing report:")
        print(message)

def send_error(error_msg):
    webhook_url = os.getenv("SLACK_WEBHOOK_URL")
    if webhook_url:
        try:
            httpx.post(webhook_url, json={"text": f"❌ Payment report error: {error_msg}"}, timeout=10)
        except:
            pass
    print(error_msg)

if __name__ == "__main__":
    main()