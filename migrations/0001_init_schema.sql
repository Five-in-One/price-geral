CREATE TABLE markets (
    id                text PRIMARY KEY,
    name              text NOT NULL,
    initials          text NOT NULL,
    latitude          double precision NOT NULL,
    longitude         double precision NOT NULL,
    primary_color     text NOT NULL,
    background_color  text NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id                 text PRIMARY KEY,
    name               text NOT NULL,
    unit               text NOT NULL,
    category           text NOT NULL,
    default_market_id  text REFERENCES markets(id),
    created_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE market_products (
    market_id   text NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    product_id  text NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    price       numeric(10,2) NOT NULL,
    PRIMARY KEY (market_id, product_id)
);

CREATE INDEX idx_products_category ON products (category);
CREATE INDEX idx_market_products_product_id ON market_products (product_id);
