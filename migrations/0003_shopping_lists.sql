-- Shopping lists people save from the app. user_id is the id price-auth
-- issued (no foreign key: that account lives in another service's database).
-- items is the list as the app sends it: [{"product": {...}, "quantity": n}].
CREATE TABLE shopping_lists (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     text NOT NULL,
    name        text NOT NULL,
    items       jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- One list per name for each person: saving under an existing name replaces it.
CREATE UNIQUE INDEX shopping_lists_user_name_idx ON shopping_lists (user_id, lower(name));
