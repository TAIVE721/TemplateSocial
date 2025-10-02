-- Añadimos la columna para la clave foránea del rol
ALTER TABLE users ADD COLUMN role_id INT;

-- Asignamos el rol de 'user' por defecto a todos los usuarios existentes
UPDATE users
SET
    role_id = (
        SELECT id
        FROM roles
        WHERE
            name = 'user'
    );

-- Hacemos que la columna no pueda ser nula y añadimos la restricción de clave foránea
ALTER TABLE users ALTER COLUMN role_id SET NOT NULL;

ALTER TABLE users
ADD CONSTRAINT fk_role FOREIGN KEY (role_id) REFERENCES roles (id);