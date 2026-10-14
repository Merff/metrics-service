CREATE TABLE metrics (
  id BIGSERIAL PRIMARY KEY,
  client_id INT NOT NULL,
  app_id INT NOT NULL,
  requests INT NOT NULL DEFAULT 0,
  landings INT NOT NULL DEFAULT 0,
  api_register_total INT NOT NULL DEFAULT 0,
  api_register_scam INT NOT NULL DEFAULT 0,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX index_metrics_on_client_id_and_app_id_and_timestamp ON metrics(client_id, app_id, timestamp);
