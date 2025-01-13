-- Add tariff plans to the products table
INSERT INTO products
(id, app_id, "name", description, price, currency, is_active, created_at, updated_at)
VALUES
    (12, 'pms', 'Basic', 'Basic plan with essential features', 5000.00, 'KZT', true, NOW(), NOW()),
    (13, 'pms', 'Pro', 'Professional plan with extended features', 10000.00, 'KZT', true, NOW(), NOW()),
    (14, 'pms', 'Expert', 'Expert plan with all features', 15000.00, 'KZT', true, NOW(), NOW());

-- Add features to the items table
INSERT INTO items (id, "name", description, topic_name, created_at, updated_at)
VALUES
-- File storage
(201, 'Storage capacity', 'Available storage space for the plan', 'storage-capacity', NOW(), NOW()),
(202, 'File upload size', 'Maximum upload file size', 'file-upload-size', NOW(), NOW()),

-- Workspaces and projects
(203, 'Workspaces', 'Management of workspaces', 'workspaces', NOW(), NOW()),
(204, 'Private projects', 'Ability to create private projects', 'private-projects', NOW(), NOW()),

-- Task board
(205, 'Task management', 'Comprehensive task management tools', 'task-management', NOW(), NOW()),
(206, 'Time tracking', 'Integration with time tracking tools', 'time-tracking', NOW(), NOW()),

-- Calendar
(207, 'Shared calendar', 'Shared team calendar', 'shared-calendar', NOW(), NOW()),
(208, 'Third-party calendar integration', 'Integration with external calendars', 'calendar-integration', NOW(), NOW()),

-- Project support
(209, 'Documentation management', 'Tools for working with project documentation', 'document-management', NOW(), NOW()),
(210, 'Support management', 'Managing user and project support', 'support-management', NOW(), NOW()),

-- Reports and analytics
(211, 'Report generation', 'Tools for creating reports', 'report-generation', NOW(), NOW()),
(212, 'Analytics', 'Built-in analytics for projects and tasks', 'analytics', NOW(), NOW()),

-- Artificial intelligence
(213, 'AI assistant', 'AI tools for automation', 'ai-assistant', NOW(), NOW());

-- Map tariff plans to features in the bundles table
INSERT INTO bundles (amount, item_id, product_id, created_at, updated_at)
VALUES
-- Basic plan
(1, 201, 12, NOW(), NOW()),
(1, 203, 12, NOW(), NOW()),
(1, 205, 12, NOW(), NOW()),
(1, 207, 12, NOW(), NOW()),

-- Pro plan
(1, 201, 13, NOW(), NOW()),
(1, 202, 13, NOW(), NOW()),
(1, 203, 13, NOW(), NOW()),
(1, 204, 13, NOW(), NOW()),
(1, 205, 13, NOW(), NOW()),
(1, 206, 13, NOW(), NOW()),
(1, 207, 13, NOW(), NOW()),
(1, 208, 13, NOW(), NOW()),
(1, 209, 13, NOW(), NOW()),
(1, 211, 13, NOW(), NOW()),

-- Expert plan
(1, 201, 14, NOW(), NOW()),
(1, 202, 14, NOW(), NOW()),
(1, 203, 14, NOW(), NOW()),
(1, 204, 14, NOW(), NOW()),
(1, 205, 14, NOW(), NOW()),
(1, 206, 14, NOW(), NOW()),
(1, 207, 14, NOW(), NOW()),
(1, 208, 14, NOW(), NOW()),
(1, 209, 14, NOW(), NOW()),
(1, 210, 14, NOW(), NOW()),
(1, 211, 14, NOW(), NOW()),
(1, 212, 14, NOW(), NOW()),
(1, 213, 14, NOW(), NOW());
