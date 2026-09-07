INSERT INTO savings_history (date_label, market_name, items_count, total_cost, savings, occurred_at) VALUES
    ('24 MAI', 'Atacado Família', 12, 187.40, 28.60, '2026-05-24'),
    ('17 MAI', 'Super Economia',  15, 215.10, 18.50, '2026-05-17'),
    ('10 MAI', 'Atacado Família', 8,  112.30, 14.20, '2026-05-10');

INSERT INTO monthly_savings (label, value, sort_order, is_active) VALUES
    ('Dez', 71,  1, false),
    ('Jan', 84,  2, false),
    ('Fev', 65,  3, false),
    ('Mar', 96,  4, false),
    ('Abr', 89,  5, false),
    ('Mai', 112, 6, true);

INSERT INTO economy_summary (total_saved, month_label, month_spend, month_savings, month_savings_percent) VALUES
    (517.10, 'Maio de 2026', 534.60, 112.30, 17.00);
