-- Add task ownership and assignment fields
-- Adds created_by, assigned_to, and updated_at columns to tasks table

ALTER TABLE tasks ADD COLUMN created_by VARCHAR(255);
ALTER TABLE tasks ADD COLUMN assigned_to VARCHAR(255);
ALTER TABLE tasks ADD COLUMN updated_at TIMESTAMP DEFAULT NOW();

-- Add foreign key constraints
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_created_by 
    FOREIGN KEY (created_by) REFERENCES users(username) ON DELETE SET NULL;

ALTER TABLE tasks ADD CONSTRAINT fk_tasks_assigned_to 
    FOREIGN KEY (assigned_to) REFERENCES users(username) ON DELETE SET NULL;

-- Add indexes for performance
CREATE INDEX idx_tasks_created_by ON tasks(created_by);
CREATE INDEX idx_tasks_assigned_to ON tasks(assigned_to);
