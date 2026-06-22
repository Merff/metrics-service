CREATE TABLE metrics (
  id BIGSERIAL PRIMARY KEY,
  client_id INT,
  app_id INT,
  requests INT,
  landings INT,
  api_register_total INT,
  api_register_scam INT,
  timestamp TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX index_metrics_on_client_id_and_app_id_and_timestamp ON metrics(client_id, app_id, timestamp);
