CREATE TABLE IF NOT EXISTS accounts(
   id serial PRIMARY KEY,
   user_id UUID NOT NULL,
   notify_upfront INT DEFAULT 7,
   notify_default_email TEXT NOT NULL,
   notify_webhook_url TEXT,
   stripe_customer_id text,
   stripe_subscription_id text,
   subscription_status text,
   slack_access_token text,
   slack_channel_id text,
   slack_webhook_url text,
   plan INT DEFAULT 0,
   FOREIGN KEY (user_id) REFERENCES auth.users (id)
);