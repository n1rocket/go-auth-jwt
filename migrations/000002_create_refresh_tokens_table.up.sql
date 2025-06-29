-- Create refresh_tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
  token UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
  revoked BOOLEAN DEFAULT FALSE,
  revoked_at TIMESTAMP WITH TIME ZONE,
  user_agent TEXT,
  ip_address INET,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at) WHERE revoked = FALSE;
CREATE INDEX idx_refresh_tokens_user_token ON refresh_tokens(user_id, token) WHERE revoked = FALSE;

-- Create trigger to update last_used_at
CREATE TRIGGER update_refresh_tokens_last_used_at BEFORE UPDATE
  ON refresh_tokens FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();