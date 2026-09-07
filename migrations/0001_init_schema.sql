CREATE TABLE savings_history (
    id           serial PRIMARY KEY,
    date_label   text NOT NULL,
    market_name  text NOT NULL,
    items_count  integer NOT NULL,
    total_cost   numeric(10,2) NOT NULL,
    savings      numeric(10,2) NOT NULL,
    occurred_at  date NOT NULL DEFAULT current_date,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE monthly_savings (
    id          serial PRIMARY KEY,
    label       text NOT NULL,
    value       numeric(10,2) NOT NULL,
    sort_order  integer NOT NULL,
    is_active   boolean NOT NULL DEFAULT false
);

CREATE TABLE economy_summary (
    id                      serial PRIMARY KEY,
    total_saved             numeric(10,2) NOT NULL,
    month_label             text NOT NULL,
    month_spend             numeric(10,2) NOT NULL,
    month_savings           numeric(10,2) NOT NULL,
    month_savings_percent   numeric(5,2) NOT NULL,
    updated_at              timestamptz NOT NULL DEFAULT now()
);
