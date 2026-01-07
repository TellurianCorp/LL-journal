-- LifeLogger LL-Journal Database Schema
-- https://api.lifelogger.life
-- company: Tellurian Corp (https://www.telluriancorp.com)
-- created in: December 2025

-- Tabela de diários
CREATE TABLE journals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_sub VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tabela de entradas
CREATE TABLE journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id UUID NOT NULL REFERENCES journals(id) ON DELETE CASCADE,
    entry_date DATE NOT NULL,
    s3_key VARCHAR(500) NOT NULL,
    git_commit_hash VARCHAR(40),
    word_count INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(journal_id, entry_date)
);

-- Tabela de versões Git
CREATE TABLE journal_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    commit_hash VARCHAR(40) NOT NULL,
    commit_message TEXT,
    author_name VARCHAR(255),
    author_email VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(entry_id, commit_hash)
);

-- Índices
CREATE INDEX idx_journals_user_sub ON journals(user_sub);
CREATE INDEX idx_entries_journal_id ON journal_entries(journal_id);
CREATE INDEX idx_entries_date ON journal_entries(entry_date);
CREATE INDEX idx_versions_entry_id ON journal_versions(entry_id);
