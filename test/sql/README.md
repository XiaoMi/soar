# Test Database

## sakila

* Download MySQL sakila database from http://downloads.mysql.com/docs/sakila-db.tar.gz .
* InnoDB added FULLTEXT support in 5.6.10, you should add version comment `/*!50610 xxx */` for `film_text` table and it's triggers.
* Merge schema and data into one file `sakila.sql`

## world\_x

world\_x contain JSON datatype, SOAR use this database for JSON testing.

* Download MySQL world\_x database from http://downloads.mysql.com/docs/world_x-db.tar.gz .
* MySQL support JSON datatype since 5.7.8, you should add version comment `/*!50708 xxx */` for `city`, `countryinfo`.
* Merge `sakila.sql`, `world_x.sql` into init.sql.

```bash
gzip init.sql
```
