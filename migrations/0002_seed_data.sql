INSERT INTO markets (id, name, initials, latitude, longitude, primary_color, background_color) VALUES
    ('af', 'Atacado Família',        'AF', -5.072984, -42.801600, '#DC2626', '#FDE8E8'),
    ('se', 'Super Economia',         'SE', -5.081092, -42.801600, '#0C63E4', '#E5F1FC'),
    ('ms', 'Mercadinho São Jorge',   'MS', -5.069380, -42.801600, '#7E22CE', '#F3E8FF'),
    ('md', 'Mercado do Bairro',      'MD', -5.085596, -42.801600, '#1F9E5F', '#E6F5ED'),
    ('hc', 'Hortifruti Central',     'HC', -5.077488, -42.801600, '#D97706', '#FDF0E2');

INSERT INTO products (id, name, unit, category, default_market_id) VALUES
    ('p1',  'Banana prata',          'kg',  'HORTIFRUTI', 'hc'),
    ('p2',  'Tomate',                'kg',  'HORTIFRUTI', 'hc'),
    ('p3',  'Alface crespa',         'un',  'HORTIFRUTI', 'hc'),
    ('p4',  'Batata inglesa',        'kg',  'HORTIFRUTI', 'af'),
    ('p5',  'Cebola',                'kg',  'HORTIFRUTI', 'af'),
    ('p6',  'Maçã gala',             'kg',  'HORTIFRUTI', 'hc'),
    ('p7',  'Arroz branco 5kg',      'pct', 'GRÃOS',      'af'),
    ('p8',  'Feijão carioca 1kg',    'pct', 'GRÃOS',      'af'),
    ('p9',  'Café torrado 500g',     'pct', 'GRÃOS',      'af'),
    ('p10', 'Óleo de soja 900ml',    'un',  'GRÃOS',      'af'),
    ('p11', 'Leite integral 1L',     'un',  'LATICÍNIOS', 'hc'),
    ('p12', 'Filé de frango',        'kg',  'CARNES',     'af');

INSERT INTO market_products (market_id, product_id, price) VALUES
    ('af', 'p1', 4.20), ('af', 'p2', 6.50), ('af', 'p3', 2.29), ('af', 'p4', 4.59),
    ('af', 'p5', 3.99), ('af', 'p6', 6.50), ('af', 'p7', 23.90), ('af', 'p8', 7.49),
    ('af', 'p9', 14.90), ('af', 'p10', 6.79), ('af', 'p11', 4.19), ('af', 'p12', 15.90),

    ('se', 'p1', 4.40), ('se', 'p2', 6.70), ('se', 'p3', 2.39), ('se', 'p4', 4.79),
    ('se', 'p5', 4.19), ('se', 'p6', 6.90), ('se', 'p7', 24.50), ('se', 'p8', 7.79),
    ('se', 'p9', 15.20), ('se', 'p10', 6.99), ('se', 'p11', 4.39), ('se', 'p12', 16.20),

    ('ms', 'p1', 4.60), ('ms', 'p2', 6.90), ('ms', 'p3', 2.49), ('ms', 'p4', 4.99),
    ('ms', 'p5', 4.39), ('ms', 'p6', 7.20), ('ms', 'p7', 25.90), ('ms', 'p8', 8.19),
    ('ms', 'p9', 15.80), ('ms', 'p10', 7.29), ('ms', 'p11', 4.49), ('ms', 'p12', 16.90),

    ('md', 'p1', 4.79), ('md', 'p2', 7.20), ('md', 'p3', 2.59), ('md', 'p4', 5.29),
    ('md', 'p5', 4.59), ('md', 'p6', 7.50), ('md', 'p7', 26.90), ('md', 'p8', 8.49),
    ('md', 'p9', 16.40), ('md', 'p10', 7.59), ('md', 'p11', 4.69), ('md', 'p12', 17.50),

    ('hc', 'p1', 4.79), ('hc', 'p2', 6.89), ('hc', 'p3', 2.49), ('hc', 'p4', 5.49),
    ('hc', 'p5', 4.79), ('hc', 'p6', 7.89), ('hc', 'p7', 27.90), ('hc', 'p8', 8.99),
    ('hc', 'p9', 16.90), ('hc', 'p10', 7.89), ('hc', 'p11', 4.59), ('hc', 'p12', 18.20);
