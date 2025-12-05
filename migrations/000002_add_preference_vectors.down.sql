ALTER TABLE profiles
DROP COLUMN IF EXISTS pref_openness,
DROP COLUMN IF EXISTS pref_conscientiousness,
DROP COLUMN IF EXISTS pref_extraversion,
DROP COLUMN IF EXISTS pref_agreeableness,
DROP COLUMN IF EXISTS pref_neuroticism;

ALTER TABLE matches
DROP COLUMN IF EXISTS match_explanation,
DROP COLUMN IF EXISTS icebreakers;
