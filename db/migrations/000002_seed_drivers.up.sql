-- Seed test drivers for development/demo
INSERT INTO reservation.drivers (id, name, phone)
VALUES
    ('driver-1', 'Budi Santoso', '+6281000000001'),
    ('driver-2', 'Siti Rahayu', '+6281000000002'),
    ('driver-3', 'Andi Pratama', '+6281000000003'),
    ('driver-4', 'Dewi Lestari', '+6281000000004')
ON CONFLICT (id) DO NOTHING;
