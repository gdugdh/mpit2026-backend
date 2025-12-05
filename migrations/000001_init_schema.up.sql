-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
-- CREATE EXTENSION IF NOT EXISTS "vector"; -- Temporarily disabled, install pgvector if needed

-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    vk_id INTEGER NOT NULL UNIQUE,
    vk_access_token TEXT,
    vk_token_expires_at TIMESTAMP WITH TIME ZONE,
    gender VARCHAR(10) NOT NULL CHECK (gender IN ('male', 'female')),
    birth_date DATE NOT NULL CHECK (birth_date <= CURRENT_DATE - INTERVAL '18 years'),
    is_verified BOOLEAN DEFAULT FALSE,
    is_online BOOLEAN DEFAULT FALSE,
    last_online_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_vk_id ON users(vk_id);
CREATE INDEX idx_users_is_online ON users(is_online);
CREATE INDEX idx_users_last_online ON users(last_online_at);

-- Profiles table
CREATE TABLE profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(100) NOT NULL,
    bio TEXT,
    city VARCHAR(100),
    interests TEXT[],
    location_lat DECIMAL(10,8),
    location_lon DECIMAL(11,8),
    location_updated_at TIMESTAMP WITH TIME ZONE,
    pref_min_age INTEGER,
    pref_max_age INTEGER,
    pref_max_distance_km INTEGER,
    is_onboarding_complete BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_profiles_user_id ON profiles(user_id);
CREATE INDEX idx_profiles_city ON profiles(city);
CREATE INDEX idx_profiles_location ON profiles USING GIST (point(location_lon, location_lat));
CREATE INDEX idx_profiles_interests ON profiles USING GIN(interests);
CREATE INDEX idx_profiles_onboarding ON profiles(is_onboarding_complete);

-- Big Five personality results table
CREATE TABLE big_five_results (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    openness DECIMAL(3,2) NOT NULL CHECK (openness >= 0 AND openness <= 1),
    conscientiousness DECIMAL(3,2) NOT NULL CHECK (conscientiousness >= 0 AND conscientiousness <= 1),
    extraversion DECIMAL(3,2) NOT NULL CHECK (extraversion >= 0 AND extraversion <= 1),
    agreeableness DECIMAL(3,2) NOT NULL CHECK (agreeableness >= 0 AND agreeableness <= 1),
    neuroticism DECIMAL(3,2) NOT NULL CHECK (neuroticism >= 0 AND neuroticism <= 1),
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_big_five_user_id ON big_five_results(user_id);
CREATE INDEX idx_big_five_extraversion ON big_five_results(extraversion);
CREATE INDEX idx_big_five_openness ON big_five_results(openness);
CREATE INDEX idx_big_five_neuroticism ON big_five_results(neuroticism);
CREATE INDEX idx_big_five_agreeableness ON big_five_results(agreeableness);
CREATE INDEX idx_big_five_conscientiousness ON big_five_results(conscientiousness);

-- User embeddings table for ML recommendations
-- Temporarily disabled - requires pgvector extension
-- Uncomment when pgvector is installed
/*
CREATE TABLE user_embeddings (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    music_vector vector(128),
    groups_vector vector(128),
    posts_vector vector(256),
    combined_vector vector(256),
    vk_data_fetched_at TIMESTAMP WITH TIME ZONE,
    vectors_updated_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_embeddings_user_id ON user_embeddings(user_id);
CREATE INDEX idx_embeddings_combined_vector ON user_embeddings USING ivfflat (combined_vector vector_cosine_ops);
*/

-- Sessions table
CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    device_info VARCHAR(255),
    ip_address VARCHAR(45),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Swipes table
CREATE TABLE swipes (
    id SERIAL PRIMARY KEY,
    swiper_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    swiped_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_like BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_swipe UNIQUE (swiper_id, swiped_id)
);

CREATE INDEX idx_swipes_swiper_id ON swipes(swiper_id, created_at DESC);
CREATE INDEX idx_swipes_swiped_id ON swipes(swiped_id, is_like);

-- Matches table
CREATE TABLE matches (
    id SERIAL PRIMARY KEY,
    user1_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_order CHECK (user1_id < user2_id),
    CONSTRAINT unique_match UNIQUE (user1_id, user2_id)
);

CREATE INDEX idx_matches_user1_id ON matches(user1_id, is_active);
CREATE INDEX idx_matches_user2_id ON matches(user2_id, is_active);
CREATE INDEX idx_matches_created_at ON matches(created_at DESC);

-- Messages table
CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    match_id INTEGER NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_match_id ON messages(match_id, created_at);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_is_read ON messages(is_read);

-- Notifications table
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id, is_read);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_profiles_updated_at BEFORE UPDATE ON profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_big_five_updated_at BEFORE UPDATE ON big_five_results
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
