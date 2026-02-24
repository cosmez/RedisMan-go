#!/usr/bin/env bash
# Populates a local Redis instance with sample keys of every type for TUI testing.
# Usage: ./scripts/seed-redis.sh [host] [port]

HOST="${1:-localhost}"
PORT="${2:-6379}"
CLI="redis-cli -h $HOST -p $PORT"

echo "Seeding Redis at $HOST:$PORT ..."

# Clear database
$CLI FLUSHDB

# --- Strings (200+) ---
echo "Adding strings..."
for i in {1..100}; do
  $CLI SET "string:key:$i" "Value for key $i with some test data"
done
for i in {1..50}; do
  $CLI SET "config:setting:$i" "config_value_$i"
done
for i in {1..50}; do
  $CLI SET "data:record:$i" "Record $i: Lorem ipsum dolor sit amet"
done
$CLI SET user:1:name "Alice Johnson"
$CLI SET user:2:name "Bob Smith"
$CLI SET user:3:name "Charlie Brown"
$CLI SET user:4:name "Diana Prince"
$CLI SET user:5:name "Eve Wilson"
$CLI SET config:app:version "1.2.3"
$CLI SET config:app:env "development"
$CLI SET config:app:debug "true"
$CLI SET config:db:host "db.example.com"
$CLI SET config:db:port "5432"
$CLI SET counter:visits 1024
$CLI SET counter:users 523
$CLI SET counter:posts 8372
$CLI SET session:abc123 '{"user":"alice","role":"admin","token":"xyz789"}'
$CLI SET session:def456 '{"user":"bob","role":"user","token":"abc123"}'
$CLI SET greeting "Welcome to RedisMan TUI!"
$CLI SET json:settings '{"theme":"dark","notifications":true,"timeout":30}'

# --- Lists (8 large lists) ---
echo "Adding lists..."
$CLI DEL queue:jobs queue:notifications email:queue messages:inbox feed:timeline queue:priority queue:background tasks:backlog

$CLI RPUSH queue:jobs \
  "job:send-email-1" "job:resize-image-1" "job:generate-report-1" "job:sync-data-1" \
  "job:cleanup-1" "job:send-email-2" "job:resize-image-2" "job:generate-report-2" \
  "job:sync-data-2" "job:cleanup-2" "job:send-email-3" "job:resize-image-3" \
  "job:generate-report-3" "job:sync-data-3" "job:cleanup-3" "job:backup-db" \
  "job:update-cache" "job:notify-users" "job:archive-logs" "job:index-search" \
  "job:compile-assets" "job:minify-js" "job:optimize-images" "job:generate-docs"

$CLI RPUSH queue:notifications \
  "Welcome to RedisMan!" "Your account was created" "Email verified successfully" \
  "Password changed" "New login from 192.168.1.1" "New login from 10.0.0.5" \
  "Payment received for order #12345" "Your order has shipped" "Delivery confirmed" \
  "Review requested" "Discount available: 20% off" "Security alert: suspicious activity" \
  "New message from admin" "System maintenance scheduled" "Backup completed successfully" \
  "Invoice #INV-2024-001" "Team member added" "Two-factor enabled" "API key rotated" \
  "Quota increased" "Support ticket resolved" "Feature released" "Maintenance completed"

for i in {1..50}; do
  $CLI RPUSH email:queue "user$i@example.com"
done

for i in {1..40}; do
  $CLI RPUSH messages:inbox "Message from user_$i: Hello, how are you?"
done

for i in {1..35}; do
  $CLI RPUSH feed:timeline "Post from author_$i at $(date -u +%s): Check out my latest article!"
done

for i in {1..25}; do
  $CLI RPUSH queue:priority "task:urgent:$i"
done

for i in {1..30}; do
  $CLI RPUSH queue:background "bg:process:$i"
done

for i in {1..45}; do
  $CLI RPUSH tasks:backlog "backlog:item:$i"
done

# --- Sets (15+ sets with many members) ---
echo "Adding sets..."
$CLI DEL tags:blog tags:products online:users visited:pages features:enabled \
  permissions:admin permissions:user skills:required countries:supported languages:supported \
  banned:ips allowed:domains categories:main status:values priority:levels

$CLI SADD tags:blog "go" "redis" "tui" "cli" "golang" "database" "tutorial" "performance" \
  "caching" "distributed" "architecture" "backend" "devops" "monitoring" "testing" "security" \
  "api" "microservices" "containers" "kubernetes"

$CLI SADD tags:products "electronics" "books" "clothing" "home" "sports" "toys" \
  "food" "furniture" "garden" "tools" "gaming" "music" "movies" "education" "fitness" \
  "outdoor" "pets" "kitchen" "office" "automotive"

$CLI SADD online:users "alice" "bob" "charlie" "diana" "eve" "frank" "grace" "henry" \
  "iris" "jack" "karen" "leo" "mary" "nora" "oscar" "patricia" "quinn" "robert" \
  "sophia" "thomas" "ursula" "victor" "wendy" "xavier" "yara" "zoe" "alex" "bailey" \
  "cameron" "drew" "emery" "finley" "gabriel" "harper"

$CLI SADD visited:pages "/" "/about" "/products" "/blog" "/contact" "/faq" "/pricing" \
  "/dashboard" "/settings" "/profile" "/search" "/categories" "/checkout" "/cart" \
  "/order-history" "/invoices" "/support" "/docs" "/api" "/tutorials" "/features" \
  "/testimonials" "/download" "/login" "/register" "/forgot-password" "/reset-password"

$CLI SADD features:enabled "dark-mode" "notifications" "two-factor" "api-access" \
  "advanced-search" "export-data" "team-collaboration" "webhooks" "sso" "audit-logs" \
  "rate-limiting" "caching" "cdn" "analytics"

$CLI SADD permissions:admin "read" "write" "delete" "manage_users" "view_analytics" \
  "manage_settings" "manage_roles" "manage_integrations" "view_audit_logs" "manage_api_keys"

$CLI SADD permissions:user "read" "write" "create_content" "comment" "share" "download"

$CLI SADD skills:required "golang" "redis" "docker" "kubernetes" "postgresql" "rest-api" \
  "git" "testing" "ci-cd" "monitoring"

$CLI SADD countries:supported "USA" "Canada" "UK" "Germany" "France" "Japan" "Australia" \
  "Brazil" "India" "Mexico" "Spain" "Italy" "Netherlands" "Sweden" "Switzerland"

$CLI SADD languages:supported "en" "es" "fr" "de" "it" "pt" "ja" "zh" "ko" "ru" "ar" "hi"

$CLI SADD banned:ips "192.168.100.50" "10.0.0.100" "172.16.0.50" "203.0.113.45" "198.51.100.89"

$CLI SADD allowed:domains "example.com" "api.example.com" "cdn.example.com" "admin.example.com" \
  "partners.example.com" "secure.example.com"

$CLI SADD categories:main "Technology" "Business" "Entertainment" "Sports" "Health" \
  "Education" "Science" "Travel" "Food" "Lifestyle"

$CLI SADD status:values "pending" "active" "inactive" "suspended" "archived" "deleted"

$CLI SADD priority:levels "critical" "high" "medium" "low" "trivial"

# --- Hashes (20+ hashes with many fields) ---
echo "Adding hashes..."
$CLI DEL user:1 user:2 user:3 user:4 user:5 user:6 user:7 user:8 product:101 product:102 \
  product:103 product:104 product:105 settings:app metrics:server api:endpoint:1 \
  api:endpoint:2 cache:stats db:replication mail:server

$CLI HSET user:1 name "Alice Johnson" email "alice@example.com" age 30 role "admin" \
  status "active" created "2023-01-15" last_login "2024-02-22" posts 150 comments 487 \
  followers 243 bio "Software engineer" country "USA" timezone "EST" verified "true"

$CLI HSET user:2 name "Bob Smith" email "bob@example.com" age 28 role "user" \
  status "active" created "2023-03-20" last_login "2024-02-21" posts 75 comments 234 \
  followers 89 bio "Data analyst" country "Canada" timezone "CST" verified "true"

$CLI HSET user:3 name "Charlie Brown" email "charlie@example.com" age 35 role "moderator" \
  status "active" created "2022-11-10" last_login "2024-02-22" posts 289 comments 891 \
  followers 512 bio "DevOps engineer" country "UK" timezone "GMT" verified "true"

$CLI HSET user:4 name "Diana Prince" email "diana@example.com" age 32 role "user" \
  status "inactive" created "2023-06-05" last_login "2024-01-15" posts 42 comments 156 \
  followers 201 bio "Product manager" country "Germany" timezone "CET" verified "false"

$CLI HSET user:5 name "Eve Wilson" email "eve@example.com" age 27 role "user" \
  status "active" created "2023-08-20" last_login "2024-02-22" posts 198 comments 643 \
  followers 334 bio "UX designer" country "Australia" timezone "AEST" verified "true"

$CLI HSET user:6 name "Frank Miller" email "frank@example.com" age 45 role "user" \
  status "active" created "2023-02-10" last_login "2024-02-20" posts 412 comments 1023 \
  followers 756 bio "System architect" country "USA" timezone "PST" verified "true"

$CLI HSET user:7 name "Grace Hopper" email "grace@example.com" age 38 role "moderator" \
  status "active" created "2023-04-25" last_login "2024-02-22" posts 234 comments 567 \
  followers 489 bio "Tech writer" country "Canada" timezone "EST" verified "true"

$CLI HSET user:8 name "Henry Ford" email "henry@example.com" age 52 role "user" \
  status "active" created "2022-12-01" last_login "2024-02-18" posts 567 comments 1234 \
  followers 987 bio "Operations manager" country "USA" timezone "CST" verified "true"

$CLI HSET product:101 name "Laptop Pro" price 1299.99 stock 45 category "electronics" \
  sku "LP-001" rating 4.8 reviews 342 color "silver" storage "512GB" ram "16GB" \
  description "High-performance laptop for professionals" brand "TechCorp" warranty "2 years"

$CLI HSET product:102 name "Wireless Mouse" price 29.99 stock 156 category "electronics" \
  sku "WM-002" rating 4.5 reviews 89 color "black" battery "1000mAh" dpi "3200" \
  description "Ergonomic wireless mouse with 2.4GHz connection" brand "PeripheralCo" warranty "1 year"

$CLI HSET product:103 name "USB-C Cable" price 9.99 stock 500 category "electronics" \
  sku "USB-003" rating 4.7 reviews 234 color "white" length "2m" power "100W" \
  description "High-speed USB-C cable for charging and data transfer" brand "CableMax" warranty "lifetime"

$CLI HSET product:104 name "4K Monitor" price 499.99 stock 28 category "electronics" \
  sku "MON-004" rating 4.9 reviews 156 color "black" resolution "3840x2160" refresh_rate "60Hz" \
  brand "DisplayPro" warranty "3 years"

$CLI HSET product:105 name "Mechanical Keyboard" price 149.99 stock 67 category "electronics" \
  sku "KEY-005" rating 4.6 reviews 203 color "white" switches "Cherry MX" backlight "RGB" \
  brand "KeyMaster" warranty "2 years"

$CLI HSET settings:app theme "dark" language "en" notifications "enabled" \
  timezone "UTC" autosave "true" backup_frequency "daily" log_level "info" \
  max_file_size "10MB" retention_days "90" compression "gzip" version "2.0.0"

$CLI HSET metrics:server cpu_usage "45.2" memory_usage "62.8" disk_usage "78.5" \
  network_in "1024.5Mbps" network_out "512.3Mbps" uptime "847h" connections "234" \
  requests_per_sec "1523" cache_hit_ratio "89.2" db_queries "5428" errors_per_min "2"

$CLI HSET api:endpoint:1 path "/api/v1/users" method "GET" rate_limit "1000/hour" \
  auth_required "true" version "1.0" response_time_ms "145" success_rate "99.8" deprecated "false"

$CLI HSET api:endpoint:2 path "/api/v1/products" method "POST" rate_limit "100/hour" \
  auth_required "true" version "1.0" response_time_ms "234" success_rate "99.5" deprecated "false"

$CLI HSET cache:stats hits 45230 misses 4567 size_mb 234.5 evictions 123 \
  ttl_avg_seconds 3600 compression_ratio "2.3" memory_efficiency "92.1" uptime_hours "720"

$CLI HSET db:replication master "primary.db.local" slave "replica1.db.local" \
  lag_ms "45" status "in_sync" last_sync "2024-02-22T10:30:00Z" transactions "1234567" \
  binlog_position "mysql-bin.000125:4567"

$CLI HSET mail:server host "smtp.example.com" port "587" encryption "tls" \
  username "noreply@example.com" auth_method "credentials" rate_limit "100/min" \
  timeout_seconds "30" queue_size "4567" delivered "234567" failed "89"

# --- Sorted Sets (15 large leaderboards/rankings) ---
echo "Adding sorted sets..."
$CLI DEL leaderboard:game scores:quiz rankings:posts rankings:comments scores:monthly \
  ratings:products ratings:comments trending:posts trending:tags stats:daily stats:weekly \
  stats:monthly queue:priority:scores waitlist:position events:trending

$CLI ZADD leaderboard:game 1500 alice 1200 bob 1800 charlie 900 diana 2100 eve 1650 frank \
  1100 grace 1950 henry 1350 iris 1750 jack 1050 karen 1600 leo 1400 mary 1850 nora \
  1550 oscar 1750 patricia 1450 quinn 1900 robert 1350 sophia

$CLI ZADD scores:quiz 95 alice 87 bob 72 charlie 100 diana 78 eve 91 frank 82 grace \
  88 henry 76 iris 94 jack 79 karen 86 leo 81 mary 93 nora 75 oscar 89 patricia \
  97 quinn 73 robert 84 sophia 92 thomas 80 ursula

for i in {1..50}; do
  score=$((RANDOM % 10000))
  $CLI ZADD rankings:posts $score "user:$i"
done

for i in {1..50}; do
  score=$((RANDOM % 5000))
  $CLI ZADD rankings:comments $score "user:$i"
done

$CLI ZADD scores:monthly 450 "alice" 380 "bob" 520 "charlie" 290 "diana" 610 "eve" \
  400 "frank" 350 "grace" 490 "henry" 310 "iris" 480 "jack" 370 "karen" 510 "leo" \
  440 "mary" 360 "nora" 530 "oscar" 300 "patricia" 470 "quinn"

$CLI ZADD ratings:products 4.8 "product:101" 4.5 "product:102" 4.7 "product:103" \
  4.9 "product:104" 4.6 "product:105" 4.2 "product:106" 4.4 "product:107" \
  4.3 "product:108" 4.7 "product:109" 4.5 "product:110"

$CLI ZADD ratings:comments 8 "comment:1" 9 "comment:2" 7 "comment:3" 10 "comment:4" \
  6 "comment:5" 8 "comment:6" 7 "comment:7" 9 "comment:8" 5 "comment:9" 8 "comment:10"

for i in {1..40}; do
  timestamp=$((RANDOM % 1000000))
  $CLI ZADD trending:posts $timestamp "post:$i"
done

for i in {1..30}; do
  count=$((RANDOM % 10000))
  $CLI ZADD trending:tags $count "tag:$i"
done

for i in {1..25}; do
  score=$((RANDOM % 1000))
  $CLI ZADD stats:daily "$score" "day:$i"
done

for i in {1..15}; do
  score=$((RANDOM % 10000))
  $CLI ZADD stats:weekly "$score" "week:$i"
done

for i in {1..12}; do
  score=$((RANDOM % 100000))
  $CLI ZADD stats:monthly "$score" "month:$i"
done

$CLI ZADD queue:priority:scores 10 "task:urgent:1" 10 "task:urgent:2" 5 "task:high:1" \
  5 "task:high:2" 3 "task:medium:1" 3 "task:medium:2" 1 "task:low:1"

for i in {1..20}; do
  position=$i
  $CLI ZADD waitlist:position $position "user:waitlist:$i"
done

for i in {1..35}; do
  score=$((RANDOM % 100000))
  $CLI ZADD events:trending $score "event:$i"
done

# --- Streams (5 large streams with many entries) ---
echo "Adding streams..."
$CLI DEL events:log audit:trail system:errors api:calls user:activity
for i in {1..60}; do
  $CLI XADD events:log '*' action "event_$i" timestamp "$(date -u +%s)" \
    user "user_$((RANDOM % 26))" ip "192.168.1.$((RANDOM % 256))" status "success" region "us-east"
done

for i in {1..50}; do
  $CLI XADD audit:trail '*' event_type "action_$i" user "admin" resource "/api/endpoint" \
    method "POST" status_code "200" duration_ms "$((RANDOM % 5000))" timestamp "$(date -u +%s)" \
    request_id "req:$i" response_size "12345"
done

for i in {1..40}; do
  severity=$((RANDOM % 3 + 1))
  $CLI XADD system:errors '*' level "level_$severity" message "Error message $i" \
    service "service:$((RANDOM % 5))" timestamp "$(date -u +%s)" stack_trace "line:$i" \
    user "user_$((RANDOM % 10))"
done

for i in {1..45}; do
  $CLI XADD api:calls '*' endpoint "/api/v1/endpoint_$((RANDOM % 10))" method "GET" \
    status_code "200" response_time_ms "$((RANDOM % 1000))" timestamp "$(date -u +%s)" \
    client_id "client:$i" auth_type "bearer"
done

for i in {1..55}; do
  activity_type=$((RANDOM % 5))
  case $activity_type in
    0) activity="login" ;;
    1) activity="logout" ;;
    2) activity="view_page" ;;
    3) activity="create_content" ;;
    4) activity="update_profile" ;;
  esac
  $CLI XADD user:activity '*' user_id "user:$i" activity "$activity" \
    timestamp "$(date -u +%s)" session_id "session:$i" ip "10.0.0.$((RANDOM % 256))"
done

# --- Keys with TTL (cache keys) ---
echo "Adding keys with TTL..."
$CLI SET cache:token:abc123 "temp-value-xyz" EX 3600
$CLI SET cache:token:def456 "temp-value-uvw" EX 1800
$CLI SET cache:token:ghi789 "temp-value-rst" EX 7200
for i in {1..50}; do
  $CLI SET "cache:session:$i" "session-data-$i" EX $((RANDOM % 3600 + 60))
done
for i in {1..30}; do
  $CLI SET "cache:page:$i" "cached-html-$i" EX $((RANDOM % 86400 + 300))
done
for i in {1..25}; do
  $CLI SET "cache:query:$i" "cached-result-$i" EX $((RANDOM % 1800 + 60))
done

# --- Various other keys ---
echo "Adding miscellaneous keys..."
$CLI SET app:version "2.0.0-beta"
$CLI SET app:name "RedisMan"
$CLI SET app:author "Developer"
for i in {1..20}; do
  $CLI SET "feature:flags:flag_$i" "enabled"
done
for i in {1..20}; do
  $CLI SET "config:option:$i" "value_$i"
done

$CLI RPUSH tasks:todo "Fix login bug" "Update documentation" "Review PR #123" "Deploy to prod"
for i in {1..15}; do
  $CLI RPUSH "tasks:sprint:$((RANDOM % 3 + 1))" "task:$i"
done

$CLI SADD permissions:admin "read" "write" "delete" "manage_users" "view_analytics"
$CLI SADD permissions:user "read" "write"
$CLI SADD permissions:guest "read"
for i in {1..10}; do
  $CLI SADD "group:members:$i" "user:1" "user:2" "user:3" "user:$((RANDOM % 10))"
done

$CLI HSET quota:limit api_calls 10000 storage_gb 100 team_members 50 projects 20
$CLI HSET quota:usage api_calls_used 4523 storage_gb_used 67 team_members_used 8 projects_used 5

$CLI ZADD priority:tasks 10 "urgent-fix" 5 "feature-request" 8 "bug-report" 3 "documentation"
for i in {1..15}; do
  $CLI ZADD "notifications:unread:user_$((RANDOM % 5))" "$i" "msg:$i"
done

# Bulk add more variety
for i in {1..30}; do
  $CLI SET "metadata:item:$i" "Item $i metadata"
done

for i in {1..20}; do
  $CLI SET "counter:metric:$i" "$((RANDOM % 10000))"
done

for i in {1..25}; do
  $CLI RPUSH "archive:batch:$i" "entry:1" "entry:2" "entry:3"
done

for i in {1..15}; do
  $CLI ZADD "scores:season:$i" "$((RANDOM % 1000))" "team:1" "$((RANDOM % 1000))" "team:2"
done

echo ""
echo "Done! Seeding complete."
$CLI DBSIZE
