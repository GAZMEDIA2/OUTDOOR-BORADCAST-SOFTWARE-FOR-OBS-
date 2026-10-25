<?php
function page_head(string $title, ?array $user = null): void {
    global $CFG;
    $f = flash();
    ?><!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title><?= e($title) ?> – Van Link</title>
<meta name="description" content="Van Link sends live video from your outside broadcast van straight to your studio over the internet, with OBS Studio. No satellite dish.">
<link rel="icon" href="assets/icon.png">
<link rel="stylesheet" href="assets/style.css">
</head>
<body>
<header class="top">
  <a class="brand" href="./"><img src="assets/icon.png" alt="" width="36" height="36"><span><b>Van Link</b><small>by <?= e($CFG['company']) ?></small></span></a>
  <nav>
    <a href="./#features">Features</a>
    <a href="./#pricing">Pricing</a>
    <?php if ($user): ?>
      <a href="account.php">My account</a>
      <?php if ($user['is_admin']): ?><a href="admin.php">Admin</a><?php endif ?>
      <a href="logout.php">Log out</a>
    <?php else: ?>
      <a href="login.php">Log in</a>
      <a class="btn small" href="signup.php">Start free trial</a>
    <?php endif ?>
  </nav>
</header>
<?php if ($f): ?><div class="flash <?= e($f[0]) ?>"><?= e($f[1]) ?></div><?php endif ?>
<?php
}

function page_foot(): void {
    global $CFG;
    ?>
<footer class="foot">
  <p>Van Link · © <?= date('Y') ?> <?= e($CFG['company']) ?> · <a href="mailto:<?= e($CFG['support_email']) ?>"><?= e($CFG['support_email']) ?></a></p>
</footer>
</body>
</html>
<?php
}
