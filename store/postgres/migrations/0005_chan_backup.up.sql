-- backup table
CREATE TABLE public.chan_backup (
  id SERIAL PRIMARY KEY,
  timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  funding_txids text,
  data text
);
