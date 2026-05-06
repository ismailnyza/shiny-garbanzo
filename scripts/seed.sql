-- Seed data for QR Restaurant
-- Password for all users: TestPass123!

-- Owner user
INSERT INTO users (id, email, password_hash, role) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'owner@test.com', '$2a$12$CkYpOOpx49YlfPpemw.04.TfC1V1x0Z6QdwUAWqEm1XqmJ6gT5Dna', 'OWNER');

-- Staff user
INSERT INTO users (id, email, password_hash, role) VALUES
  ('b0000000-0000-0000-0000-000000000001', 'staff@test.com', '$2a$12$CkYpOOpx49YlfPpemw.04.TfC1V1x0Z6QdwUAWqEm1XqmJ6gT5Dna', 'STAFF');

-- Restaurant
INSERT INTO restaurants (id, owner_id, name, slug, description, address, phone) VALUES
  ('c0000000-0000-0000-0000-000000000001',
   'a0000000-0000-0000-0000-000000000001',
   'Pizza Palace',
   'pizza-palace',
   'Best pizza in town',
   '123 Main St, Colombo',
   '+94771234567');

-- Restaurant staff assignment
INSERT INTO restaurant_staff (id, restaurant_id, user_id) VALUES
  ('d0000000-0000-0000-0000-000000000001',
   'c0000000-0000-0000-0000-000000000001',
   'b0000000-0000-0000-0000-000000000001');

-- Theme
INSERT INTO themes (id, restaurant_id, primary_color, secondary_color) VALUES
  ('e0000000-0000-0000-0000-000000000001',
   'c0000000-0000-0000-0000-000000000001',
   '#FF6B35',
   '#F7C59F');

-- Tables
INSERT INTO tables (id, restaurant_id, table_code, qr_token) VALUES
  ('f0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'T01', 'qr-demo-table-t01-000000000001'),
  ('f0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'T02', 'qr-demo-table-t02-000000000002');

-- Menu Categories
INSERT INTO menu_categories (id, restaurant_id, name, sort_order) VALUES
  ('g0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'Starters', 1),
  ('g0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'Mains', 2),
  ('g0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 'Drinks', 3);

-- Menu Items
INSERT INTO menu_items (id, category_id, restaurant_id, name, description, price_cents, sort_order) VALUES
  ('h0000000-0000-0000-0000-000000000001', 'g0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'Garlic Bread', 'Crispy garlic bread with herb butter', 400, 1),
  ('h0000000-0000-0000-0000-000000000002', 'g0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'Bruschetta', 'Tomato, basil, and mozzarella on toasted bread', 600, 2),
  ('h0000000-0000-0000-0000-000000000003', 'g0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'Margherita Pizza', 'Classic tomato sauce and mozzarella', 1200, 1),
  ('h0000000-0000-0000-0000-000000000004', 'g0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'Pepperoni Pizza', 'Pepperoni, tomato sauce, and mozzarella', 1500, 2),
  ('h0000000-0000-0000-0000-000000000005', 'g0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'Pasta Carbonara', 'Creamy carbonara with pancetta', 1100, 3),
  ('h0000000-0000-0000-0000-000000000006', 'g0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 'Lemonade', 'Freshly squeezed lemonade', 300, 1),
  ('h0000000-0000-0000-0000-000000000007', 'g0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 'Iced Tea', 'Homemade iced tea with peach', 350, 2);
