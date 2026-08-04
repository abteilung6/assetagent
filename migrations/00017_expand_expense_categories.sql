-- +goose Up
-- +goose StatementBegin
INSERT INTO categories (slug, display_name, kind, is_system) VALUES
    ('dining', 'Restaurant & Café', 'expense', true),
    ('travel', 'Reisen', 'expense', true),
    ('shopping', 'Shopping', 'expense', true),
    ('subscriptions', 'Abos', 'expense', true),
    ('personal_care', 'Drogerie & Pflege', 'expense', true),
    ('education', 'Bildung', 'expense', true),
    ('children', 'Kinder & Familie', 'expense', true),
    ('pets', 'Haustiere', 'expense', true),
    ('taxes_fees', 'Steuern & Gebühren', 'expense', true),
    ('cash_atm', 'Bargeld', 'expense', true)
ON CONFLICT (slug) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM categories
WHERE slug IN (
    'dining',
    'travel',
    'shopping',
    'subscriptions',
    'personal_care',
    'education',
    'children',
    'pets',
    'taxes_fees',
    'cash_atm'
)
AND is_system = true;
-- +goose StatementEnd
