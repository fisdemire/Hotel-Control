-- +goose Up

CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO users (email, password_hash, role)
VALUES ('admin@hotel.local', crypt('admin123', gen_salt('bf', 10)), 'admin');

INSERT INTO hotels (name, address)
VALUES ('Grand Hotel', 'Main street 1');

INSERT INTO room_types (hotel_id, name, capacity, price_per_night)
SELECT h.id, v.name, v.capacity, v.price
FROM hotels h,
     (VALUES
        ('Standard', 2,  80.00),
        ('Deluxe',   3, 130.00),
        ('Family',   4, 170.00)
     ) AS v(name, capacity, price)
WHERE h.name = 'Grand Hotel';

INSERT INTO rooms (hotel_id, room_type_id, number, status)
SELECT rt.hotel_id, rt.id, v.number, 'active'
FROM room_types rt
JOIN hotels h ON h.id = rt.hotel_id
JOIN (VALUES
        ('101', 'Standard'),
        ('102', 'Standard'),
        ('201', 'Deluxe'),
        ('202', 'Deluxe'),
        ('301', 'Family')
     ) AS v(number, type_name) ON v.type_name = rt.name
WHERE h.name = 'Grand Hotel';

-- +goose Down

DELETE FROM rooms
WHERE hotel_id IN (SELECT id FROM hotels WHERE name = 'Grand Hotel');

DELETE FROM room_types
WHERE hotel_id IN (SELECT id FROM hotels WHERE name = 'Grand Hotel');

DELETE FROM hotels WHERE name = 'Grand Hotel';

DELETE FROM users WHERE email = 'admin@hotel.local';