-- Add preference columns to profiles table for Reinforcement Learning
ALTER TABLE profiles
ADD COLUMN pref_openness DECIMAL(3,2) CHECK (pref_openness >= 0 AND pref_openness <= 1),
ADD COLUMN pref_conscientiousness DECIMAL(3,2) CHECK (pref_conscientiousness >= 0 AND pref_conscientiousness <= 1),
ADD COLUMN pref_extraversion DECIMAL(3,2) CHECK (pref_extraversion >= 0 AND pref_extraversion <= 1),
ADD COLUMN pref_agreeableness DECIMAL(3,2) CHECK (pref_agreeableness >= 0 AND pref_agreeableness <= 1),
ADD COLUMN pref_neuroticism DECIMAL(3,2) CHECK (pref_neuroticism >= 0 AND pref_neuroticism <= 1);

-- Add columns for Gemini AI results to matches table
ALTER TABLE matches
ADD COLUMN match_explanation TEXT,
ADD COLUMN icebreakers TEXT[];

-- Create index for preference queries if needed (optional for now)
