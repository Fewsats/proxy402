import psycopg2
import psycopg2.extras
from datetime import datetime, timedelta
import os
import httpx
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

# Environment Variables
SLACK_WEBHOOK_URL = os.environ.get("SLACK_WEBHOOK_URL")
DB_HOST = os.environ.get("DB_HOST", "localhost")
DB_PORT = os.environ.get("DB_PORT", "5432")
DB_USER = os.environ.get("DB_USER")
DB_PASSWORD = os.environ.get("DB_PASSWORD")
DB_NAME = os.environ.get("DB_NAME")

# SQL query for filtered stats (excluding @fewsats.com users)
SQL_QUERY_FILTERED_STATS = """
SELECT
    to_char(p.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS purchase_date_str,
    COUNT(*) AS total_payments_on_day,
    COALESCE(SUM(p.price), 0) AS total_amount_on_day,
    COUNT(CASE WHEN p.is_test = true THEN 1 END) AS test_payments_count,
    COALESCE(SUM(CASE WHEN p.is_test = true THEN p.price ELSE 0 END), 0) AS test_payments_total_amount,
    COUNT(CASE WHEN p.is_test = false THEN 1 END) AS live_payments_count,
    COALESCE(SUM(CASE WHEN p.is_test = false THEN p.price ELSE 0 END), 0) AS live_payments_total_amount
FROM
    purchases p
JOIN
    paid_routes pr ON p.paid_route_id = pr.id
JOIN
    users u ON pr.user_id = u.id
WHERE
    u.email NOT LIKE '%%@fewsats.com'
GROUP BY
    purchase_date_str
ORDER BY
    purchase_date_str DESC;
"""

# SQL query for unfiltered stats (all users)
SQL_QUERY_UNFILTERED_STATS = """
SELECT
    to_char(p.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS purchase_date_str,
    COUNT(*) AS total_payments_on_day,
    COALESCE(SUM(p.price), 0) AS total_amount_on_day,
    COUNT(CASE WHEN p.is_test = true THEN 1 END) AS test_payments_count,
    COALESCE(SUM(CASE WHEN p.is_test = true THEN p.price ELSE 0 END), 0) AS test_payments_total_amount,
    COUNT(CASE WHEN p.is_test = false THEN 1 END) AS live_payments_count,
    COALESCE(SUM(CASE WHEN p.is_test = false THEN p.price ELSE 0 END), 0) AS live_payments_total_amount
FROM
    purchases p
GROUP BY
    purchase_date_str
ORDER BY
    purchase_date_str DESC;
"""

def fetch_payment_stats(query):
    """Generic function to fetch payment stats using the given query"""
    conn = None
    if not all([DB_USER, DB_PASSWORD, DB_NAME]):
        print("Error: DB_USER, DB_PASSWORD, or DB_NAME environment variables not set.")
        return None
    try:
        conn = psycopg2.connect(
            host=DB_HOST, port=DB_PORT, user=DB_USER, password=DB_PASSWORD, dbname=DB_NAME
        )
        with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cur:
            cur.execute(query)
            rows = cur.fetchall()
        data = []
        for row_dict in rows:
            data.append({
                'purchase_date': row_dict['purchase_date_str'],
                'purchase_date_dt': datetime.strptime(row_dict['purchase_date_str'], '%Y-%m-%d').date(),
                'total_payments_on_day': int(row_dict['total_payments_on_day']),
                'total_amount_on_day': int(row_dict['total_amount_on_day']), # Smallest unit of USDC (10^-6)
                'test_payments_count': int(row_dict['test_payments_count']),
                'test_payments_total_amount': int(row_dict['test_payments_total_amount']), # Smallest unit of USDC
                'live_payments_count': int(row_dict['live_payments_count']),
                'live_payments_total_amount': int(row_dict['live_payments_total_amount']) # Smallest unit of USDC
            })
        return data
    except psycopg2.Error as e:
        print(f"Database error: {e}")
        return None
    finally:
        if conn: conn.close()

def fetch_filtered_stats():
    """Fetch payment stats excluding @fewsats.com users"""
    return fetch_payment_stats(SQL_QUERY_FILTERED_STATS)

def fetch_unfiltered_stats():
    """Fetch payment stats for all users"""
    return fetch_payment_stats(SQL_QUERY_UNFILTERED_STATS)

def val_to_usdc_str(value_smallest_unit):
    if value_smallest_unit is None: return "N/A"
    usdc_value = value_smallest_unit / 1_000_000.0
    # Format to 6 decimal places, then remove trailing zeros and unnecessary decimal point
    formatted_value = f"{usdc_value:.6f}".rstrip('0').rstrip('.')
    return formatted_value

def format_delta(delta_smallest_unit, is_volume=False):
    if delta_smallest_unit is None: return ""
    prefix = '+' if delta_smallest_unit >= 0 else ''
    if is_volume:
        return f" ({prefix}{val_to_usdc_str(delta_smallest_unit)} USDC)" # Add USDC unit
    return f" ({prefix}{delta_smallest_unit})"

def calculate_stats(data_rows):
    stats = {
        "live_tx_today": 0, "live_tx_delta_str": "",
        "test_tx_today": 0, "test_tx_delta_str": "",
        "live_vol_today_usdc_str": val_to_usdc_str(0), "live_vol_delta_usdc_str": "",
        "test_vol_today_usdc_str": val_to_usdc_str(0), "test_vol_delta_usdc_str": "",
        "weekly_live_tx": 0, "weekly_live_vol_usdc_str": val_to_usdc_str(0),
        "weekly_test_tx": 0, "weekly_test_vol_usdc_str": val_to_usdc_str(0),
        "has_today_data": False
    }

    if not data_rows: return stats

    today_data = data_rows[0]
    stats["has_today_data"] = True
    stats["live_tx_today"] = today_data['live_payments_count']
    stats["test_tx_today"] = today_data['test_payments_count']
    stats["live_vol_today_usdc_str"] = val_to_usdc_str(today_data['live_payments_total_amount'])
    stats["test_vol_today_usdc_str"] = val_to_usdc_str(today_data['test_payments_total_amount'])

    yesterday_data = data_rows[1] if len(data_rows) > 1 else None
    if yesterday_data and (today_data['purchase_date_dt'] - yesterday_data['purchase_date_dt']).days == 1:
        live_tx_delta = today_data['live_payments_count'] - yesterday_data['live_payments_count']
        stats["live_tx_delta_str"] = format_delta(live_tx_delta)
        
        test_tx_delta = today_data['test_payments_count'] - yesterday_data['test_payments_count']
        stats["test_tx_delta_str"] = format_delta(test_tx_delta)

        live_vol_delta = today_data['live_payments_total_amount'] - yesterday_data['live_payments_total_amount']
        stats["live_vol_delta_usdc_str"] = format_delta(live_vol_delta, is_volume=True)

        test_vol_delta = today_data['test_payments_total_amount'] - yesterday_data['test_payments_total_amount']
        stats["test_vol_delta_usdc_str"] = format_delta(test_vol_delta, is_volume=True)

    today_date_obj = today_data['purchase_date_dt']
    start_of_week = today_date_obj - timedelta(days=6)
    
    weekly_live_tx_sum, weekly_live_vol_sum_smallest_unit = 0, 0
    weekly_test_tx_sum, weekly_test_vol_sum_smallest_unit = 0, 0

    for row in data_rows:
        if start_of_week <= row['purchase_date_dt'] <= today_date_obj:
            weekly_live_tx_sum += row['live_payments_count']
            weekly_live_vol_sum_smallest_unit += row['live_payments_total_amount']
            weekly_test_tx_sum += row['test_payments_count']
            weekly_test_vol_sum_smallest_unit += row['test_payments_total_amount']
            
    stats["weekly_live_tx"] = weekly_live_tx_sum
    stats["weekly_live_vol_usdc_str"] = val_to_usdc_str(weekly_live_vol_sum_smallest_unit)
    stats["weekly_test_tx"] = weekly_test_tx_sum
    stats["weekly_test_vol_usdc_str"] = val_to_usdc_str(weekly_test_vol_sum_smallest_unit)
    
    return stats

def format_combined_slack_message(filtered_stats, unfiltered_stats):
    if not filtered_stats["has_today_data"] and not unfiltered_stats["has_today_data"]:
        return {
            "blocks": [{
                "type": "section",
                "text": { "type": "mrkdwn", "text": "📈 *Daily Payment Stats*\n- No transaction data for today."}
            }]
        }

    blocks = [
        {
            "type": "header",
            "text": {"type": "plain_text", "text": "📈 Daily Payment Stats", "emoji": True}
        },
        # Side-by-side column headers
        {
            "type": "section",
            "fields": [
                {"type": "mrkdwn", "text": "*All Users (Unfiltered)*"},
                {"type": "mrkdwn", "text": "*External Users Only*\n*(Excluding @fewsats.com)*"}
            ]
        },
        # Transactions (side by side)
        {
            "type": "section",
            "fields": [
                {"type": "mrkdwn", "text": f"*Live Txns:* `{unfiltered_stats['live_tx_today']}`{unfiltered_stats['live_tx_delta_str']}\n*Test Txns:* `{unfiltered_stats['test_tx_today']}`{unfiltered_stats['test_tx_delta_str']}"},
                {"type": "mrkdwn", "text": f"*Live Txns:* `{filtered_stats['live_tx_today']}`{filtered_stats['live_tx_delta_str']}\n*Test Txns:* `{filtered_stats['test_tx_today']}`{filtered_stats['test_tx_delta_str']}"}
            ]
        },
        # Volumes (side by side)
        {
            "type": "section",
            "fields": [
                {"type": "mrkdwn", "text": f"*Live Vol:* `{unfiltered_stats['live_vol_today_usdc_str']}`{unfiltered_stats['live_vol_delta_usdc_str']}\n*Test Vol:* `{unfiltered_stats['test_vol_today_usdc_str']}`{unfiltered_stats['test_vol_delta_usdc_str']}"},
                {"type": "mrkdwn", "text": f"*Live Vol:* `{filtered_stats['live_vol_today_usdc_str']}`{filtered_stats['live_vol_delta_usdc_str']}\n*Test Vol:* `{filtered_stats['test_vol_today_usdc_str']}`{filtered_stats['test_vol_delta_usdc_str']}"}
            ]
        },
        # Weekly (side by side)
        {
            "type": "section",
            "fields": [
                {"type": "mrkdwn", "text": f"*Weekly (All Users):*\n• Live: `{unfiltered_stats['weekly_live_tx']}` txns (`{unfiltered_stats['weekly_live_vol_usdc_str']}` USDC)\n• Test: `{unfiltered_stats['weekly_test_tx']}` txns (`{unfiltered_stats['weekly_test_vol_usdc_str']}` USDC)"},
                {"type": "mrkdwn", "text": f"*Weekly (External Only):*\n• Live: `{filtered_stats['weekly_live_tx']}` txns (`{filtered_stats['weekly_live_vol_usdc_str']}` USDC)\n• Test: `{filtered_stats['weekly_test_tx']}` txns (`{filtered_stats['weekly_test_vol_usdc_str']}` USDC)"}
            ]
        }
    ]
    return {"blocks": blocks}

def send_slack_message(payload):
    if not SLACK_WEBHOOK_URL:
        print("SLACK_WEBHOOK_URL not set. Cannot send message.")
        return False
    try:
        # Send the raw payload which is now expected to be a dict for Block Kit
        response = httpx.post(SLACK_WEBHOOK_URL, json=payload, timeout=10)
        response.raise_for_status()
        print("Slack message sent successfully.")
        return True
    except httpx.RequestError as e:
        print(f"Error sending Slack message: {e}")
        return False
    except httpx.HTTPStatusError as e:
        print(f"Error sending Slack message: {e.response.status_code} - {e.response.text}")
        return False

if __name__ == "__main__":
    print(f"Running daily payment stats report at {datetime.now()}...")
    
    # Fetch both filtered and unfiltered stats
    filtered_data = fetch_filtered_stats()
    unfiltered_data = fetch_unfiltered_stats()
    
    if filtered_data is None or unfiltered_data is None:
        error_payload = {
            "blocks": [{
                "type": "section",
                "text": { "type": "mrkdwn", "text": "❌ *Error*\nFailed to retrieve payment stats data from DB."}
            }]
        }
        print("Failed to retrieve payment stats data from DB.")
        send_slack_message(error_payload)
        exit(1)

    # Calculate stats for both datasets
    filtered_stats = calculate_stats(filtered_data)
    unfiltered_stats = calculate_stats(unfiltered_data)
    
    # Create combined message
    slack_payload = format_combined_slack_message(filtered_stats, unfiltered_stats)
    
    # For local checking
    import json
    print(f"Formatted payload:\n{json.dumps(slack_payload, indent=2)}")
    
    # Send to Slack
    send_slack_message(slack_payload)
    print("Report finished.") 