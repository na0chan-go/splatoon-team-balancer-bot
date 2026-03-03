UPDATE player_stats
SET rating_delta = rating
WHERE rating_delta = 0 AND rating != 0;

