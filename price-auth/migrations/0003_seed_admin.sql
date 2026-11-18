-- Bootstrap admin so there's a first account to log into the Admin panel
-- with. Password is "TrocarSenha123!" — change it in a real deployment;
-- this is seed data for local/dev use only.
INSERT INTO users (username, password_hash, role)
VALUES ('admin', '$2a$10$HlbzvTfiWT3mVPzQRlUQMO9pFz1SKoMFL0ZoNSvU0UOwYtkehhN5a', 'admin');
