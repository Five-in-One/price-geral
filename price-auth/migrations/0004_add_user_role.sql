-- Third role: 'user', for people who sign up in the mobile app through
-- POST /auth/signup (saved shopping lists live in price-user, keyed by this
-- account). It has no access to the Admin panel or the Company page.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'company', 'user'));
