-- Distinguishes internal Price staff (Admin panel) from partner markets
-- (Company page). Public self-registration always gets 'company' — only
-- an existing admin can create another admin account, via
-- POST /auth/admin/users.
ALTER TABLE users ADD COLUMN role text NOT NULL DEFAULT 'company';
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'company'));
