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

SQL_QUERY_PAYMENT_STATS = """
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

def fetch_payment_stats_from_db():
    conn = None
    if not all([DB_USER, DB_PASSWORD, DB_NAME]):
        print("Error: DB_USER, DB_PASSWORD, or DB_NAME environment variables not set.")
        return None
    try:
        conn = psycopg2.connect(
            host=DB_HOST, port=DB_PORT, user=DB_USER, password=DB_PASSWORD, dbname=DB_NAME
        )
        with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cur:
            cur.execute(SQL_QUERY_PAYMENT_STATS)
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

def val_to_usdc_str(value_smallest_unit):
    if value_smallest_unit is None: return "N/A"
    usdc_value = value_smallest_unit / 1_000_000.0
    return f"{usdc_value:.2f}" # Display with 2 decimal places for readability

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

def format_slack_message_blocks(s):
    if not s["has_today_data"]:
        return {
            "blocks": [{
                "type": "section",
                "text": { "type": "mrkdwn", "text": "📈 *Daily Payment Stats (Live / Test)*\n- No transaction data for today."}
            }]
        }

    blocks = [
        {
            "type": "header",
            "text": {"type": "plain_text", "text": "📈 Daily Payment Stats (Live / Test)", "emoji": True}
        },
        {
            "type": "section",
            "fields": [
                {"type": "mrkdwn", "text": f"*Live Transactions:*\n`{s['live_tx_today']}`{s['live_tx_delta_str']}"},
                {"type": "mrkdwn", "text": f"*Test Transactions:*\n`{s['test_tx_today']}`{s['test_tx_delta_str']}"}
            ]
        },
        {
            "type": "section",
            "fields": [
                {"type": "mrkdwn", "text": f"*Live Volume (USDC):*\n`{s['live_vol_today_usdc_str']}`{s['live_vol_delta_usdc_str']}"},
                {"type": "mrkdwn", "text": f"*Test Volume (USDC):*\n`{s['test_vol_today_usdc_str']}`{s['test_vol_delta_usdc_str']}"}
            ]
        },
        {"type": "divider"},
        {
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": "*Weekly Cumulative (Live / Test):*\n"
                        f"• *Live:* `{s['weekly_live_tx']}` transactions (`{s['weekly_live_vol_usdc_str']}` USDC)\n"
                        f"• *Test:* `{s['weekly_test_tx']}` transactions (`{s['weekly_test_vol_usdc_str']}` USDC)"
            }
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
    fetched_data = fetch_payment_stats_from_db()
    
    if fetched_data is None:
        error_payload = format_slack_message_blocks({"has_today_data": False, "error_message": "Failed to retrieve payment stats data from DB."})
        print("Failed to retrieve payment stats data from DB.")
        send_slack_message(error_payload)
        exit(1)

    calculated_values = calculate_stats(fetched_data)
    slack_payload = format_slack_message_blocks(calculated_values)
    
    # For local checking, you might want to print the JSON or a summary
    import json
    print(f"Formatted payload:\n{json.dumps(slack_payload, indent=2)}")
    send_slack_message(slack_payload)
    print("Report finished.") 