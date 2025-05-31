-- Seed data for development
-- This file is used to populate the database with sample data
-- Insert sample users
INSERT INTO users (name, email, age)
VALUES ('John Doe', 'john@example.com', 30),
    ('Jane Smith', 'jane@example.com', 25),
    ('Bob Johnson', 'bob@example.com', 35),
    ('Alice Brown', 'alice@example.com', 28),
    ('Charlie Wilson', 'charlie@example.com', 32) ON CONFLICT (email) DO NOTHING;
-- Insert sample categories
INSERT INTO categories (name, description, color)
VALUES (
        'Technology',
        'Posts about technology and programming',
        '#3B82F6'
    ),
    (
        'Lifestyle',
        'Posts about lifestyle and personal experiences',
        '#10B981'
    ),
    (
        'Business',
        'Posts about business and entrepreneurship',
        '#F59E0B'
    ),
    (
        'Health',
        'Posts about health and wellness',
        '#EF4444'
    ),
    (
        'Travel',
        'Posts about travel and adventures',
        '#8B5CF6'
    ) ON CONFLICT (name) DO NOTHING;
-- Insert sample tags
INSERT INTO tags (name, slug)
VALUES ('golang', 'golang'),
    ('react', 'react'),
    ('typescript', 'typescript'),
    ('database', 'database'),
    ('web-development', 'web-development'),
    ('tutorial', 'tutorial'),
    ('tips', 'tips'),
    ('beginner', 'beginner'),
    ('advanced', 'advanced'),
    ('productivity', 'productivity') ON CONFLICT (name) DO NOTHING;
-- Insert sample posts
INSERT INTO posts (title, content, user_id, published)
VALUES (
        'Getting Started with GoFlux',
        'GoFlux is a modern full-stack framework that combines the power of Go with the flexibility of React. In this post, we''ll explore how to get started with building your first application using GoFlux.',
        1,
        true
    ),
    (
        'Building Type-Safe APIs',
        'One of the key features of GoFlux is automatic type generation. This ensures that your frontend and backend stay in sync, reducing bugs and improving developer experience.',
        2,
        true
    ),
    (
        'Database Management with SQLC',
        'SQLC is a fantastic tool for generating type-safe Go code from SQL queries. In this post, we''ll explore how to set up and use SQLC in your GoFlux projects.',
        1,
        true
    ),
    (
        'React Query Integration',
        'Learn how to integrate React Query with your GoFlux application for powerful data fetching, caching, and synchronization.',
        3,
        true
    ),
    (
        'Draft: Advanced GoFlux Patterns',
        'This is a draft post about advanced patterns and best practices when working with GoFlux applications.',
        2,
        false
    ),
    (
        'Deployment Strategies',
        'Explore different strategies for deploying your GoFlux applications to production, including Docker, cloud platforms, and traditional servers.',
        4,
        true
    ) ON CONFLICT DO NOTHING;
-- Insert sample comments
INSERT INTO comments (content, post_id, user_id)
VALUES (
        'Great introduction! This really helped me understand the basics.',
        1,
        2
    ),
    (
        'Thanks for sharing this. Looking forward to more GoFlux content.',
        1,
        3
    ),
    (
        'The type safety features are amazing. No more runtime errors!',
        2,
        1
    ),
    (
        'SQLC has been a game-changer for our team. Highly recommended.',
        3,
        4
    ),
    (
        'React Query makes data fetching so much easier. Great tutorial!',
        4,
        1
    ),
    (
        'When will this draft be published? Sounds very interesting.',
        5,
        3
    ),
    (
        'Deployment can be tricky. This guide covers all the important aspects.',
        6,
        2
    ) ON CONFLICT DO NOTHING;
-- Insert sample user profiles
INSERT INTO user_profiles (user_id, bio, website, location)
VALUES (
        1,
        'Full-stack developer passionate about Go and React. Creator of GoFlux.',
        'https://johndoe.dev',
        'San Francisco, CA'
    ),
    (
        2,
        'Frontend specialist with expertise in React and TypeScript.',
        'https://janesmith.com',
        'New York, NY'
    ),
    (
        3,
        'Backend engineer focused on scalable systems and databases.',
        NULL,
        'Austin, TX'
    ),
    (
        4,
        'DevOps engineer and cloud architecture enthusiast.',
        'https://alicebrown.tech',
        'Seattle, WA'
    ) ON CONFLICT (user_id) DO NOTHING;
-- Link posts to categories
INSERT INTO post_categories (post_id, category_id)
VALUES (1, 1),
    -- Getting Started with GoFlux -> Technology
    (2, 1),
    -- Building Type-Safe APIs -> Technology
    (3, 1),
    -- Database Management with SQLC -> Technology
    (4, 1),
    -- React Query Integration -> Technology
    (5, 1),
    -- Draft: Advanced GoFlux Patterns -> Technology
    (6, 3) -- Deployment Strategies -> Business
    ON CONFLICT DO NOTHING;
-- Link posts to tags
INSERT INTO post_tags (post_id, tag_id)
VALUES (1, 1),
    -- Getting Started with GoFlux -> golang
    (1, 2),
    -- Getting Started with GoFlux -> react
    (1, 6),
    -- Getting Started with GoFlux -> tutorial
    (1, 8),
    -- Getting Started with GoFlux -> beginner
    (2, 1),
    -- Building Type-Safe APIs -> golang
    (2, 3),
    -- Building Type-Safe APIs -> typescript
    (2, 5),
    -- Building Type-Safe APIs -> web-development
    (3, 1),
    -- Database Management with SQLC -> golang
    (3, 4),
    -- Database Management with SQLC -> database
    (3, 6),
    -- Database Management with SQLC -> tutorial
    (4, 2),
    -- React Query Integration -> react
    (4, 3),
    -- React Query Integration -> typescript
    (4, 6),
    -- React Query Integration -> tutorial
    (5, 1),
    -- Draft: Advanced GoFlux Patterns -> golang
    (5, 9),
    -- Draft: Advanced GoFlux Patterns -> advanced
    (6, 1),
    -- Deployment Strategies -> golang
    (6, 7),
    -- Deployment Strategies -> tips
    (6, 10) -- Deployment Strategies -> productivity
    ON CONFLICT DO NOTHING;