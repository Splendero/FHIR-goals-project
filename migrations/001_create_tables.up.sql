CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE patients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    active BOOLEAN NOT NULL DEFAULT true,
    name_family VARCHAR(255),
    name_given TEXT[],
    gender VARCHAR(20),
    birth_date DATE,
    identifier_system VARCHAR(255),
    identifier_value VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE goals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    lifecycle_status VARCHAR(50) NOT NULL DEFAULT 'proposed',
    achievement_status VARCHAR(50) NOT NULL DEFAULT 'in-progress',
    category_code VARCHAR(100),
    category_display VARCHAR(255),
    priority VARCHAR(20),
    description_text TEXT NOT NULL,
    subject_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    target_measure_code VARCHAR(100),
    target_measure_display VARCHAR(255),
    target_detail_value DOUBLE PRECISION,
    target_detail_unit VARCHAR(50),
    target_due_date DATE,
    start_date DATE,
    status_date DATE,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE care_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    intent VARCHAR(50) NOT NULL DEFAULT 'plan',
    title VARCHAR(255) NOT NULL,
    description TEXT,
    subject_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    period_start DATE,
    period_end DATE,
    goal_ids UUID[] DEFAULT '{}',
    category_code VARCHAR(100),
    category_display VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE observations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    status VARCHAR(50) NOT NULL DEFAULT 'final',
    category_code VARCHAR(100),
    category_display VARCHAR(255),
    code_code VARCHAR(100) NOT NULL,
    code_display VARCHAR(255),
    subject_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    effective_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    value_quantity_value DOUBLE PRECISION,
    value_quantity_unit VARCHAR(50),
    value_quantity_code VARCHAR(50),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_goals_subject ON goals(subject_id);
CREATE INDEX idx_goals_status ON goals(lifecycle_status);
CREATE INDEX idx_goals_category ON goals(category_code);
CREATE INDEX idx_care_plans_subject ON care_plans(subject_id);
CREATE INDEX idx_observations_subject ON observations(subject_id);
CREATE INDEX idx_observations_code ON observations(code_code);
CREATE INDEX idx_observations_date ON observations(effective_date);
