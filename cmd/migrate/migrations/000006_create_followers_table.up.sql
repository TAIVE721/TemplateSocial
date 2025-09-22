CREATE TABLE IF NOT EXISTS followers (
  -- El ID del usuario que es seguido
  user_id bigint NOT NULL,
  -- El ID del usuario que está siguiendo
  follower_id bigint NOT NULL,
  created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),

-- La clave primaria es la combinación de ambos IDs para evitar duplicados
PRIMARY KEY (user_id, follower_id),

-- Claves foráneas para mantener la integridad de los datos
FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (follower_id) REFERENCES users (id) ON DELETE CASCADE
);