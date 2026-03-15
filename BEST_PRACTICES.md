# Best practices

This is unfortunate, but since 2018 many things were changed. Most of them
became way worse. Previous iterations of censorship systems were very dumb,
DPI were primitive and filtered very obvious things. Nowadays they are
way more intelligent and it is very naive to treat them frivolously.

In 2026 is not enough to pretend that your mtg installation is a Microsoft
website that sits in Amsterdam Digital Ocean location. Now your installation
has to be a website that is mtg in disguise. Yes, it requires a bit more effort
but this effort is probably less than rotating proxies each other day.

mtproto traffic, even with FakeTLS, has its specifics that are probably
very well known by DPI systems. These specifics are not something unique but
could mark an IP address as suspicious. Now let's think:

1. You have a proxy in Amsterdam Digital Ocean that tells it is microsoft.com
   how hard could it be to find out that this is probably fake? 1 or probably 2
   DNS queries for `microsoft.com`? In case of some CDN, there are ECS-powered
   resolvers that are very capable to return results from POV of some subnets.
   If censor sees no relevant results, will they be afraid to block IP?
2. You have a proxy in Amsterdam Digital Ocean that tells it is a website from
   the same public subnet. But not the same. Would it be hard to make these DNS
   queries and ban IP?

The correct way of having this proxy is following:

1. Register a domain name
2. Get some VPS, probably in your domestic location
3. Set that domain name from a step 1 to IP address of that VPS
4. Generate a couple of HTML pages by LLMs or even copy them from elsewhere
5. Set some webserver and issue TLS certificates with Let's Encrypt or any other
   name
6. Set mtg before this webserver.
7. Use sing-box or anything like that to provide local socks5 interface and
   have VPNized uplinks
8. Set up mtg to use socks5 from a 7 step.

In that case you will get a match of DNS and SNI in requests. As a side effect,
your proxy will work with XTLS and its friends: XTLS in sniff mode ignores
IP address a client wants to connect to. Instead, it reads SNI and connect
to resolved address: a clever idea if user does not have a trustworthy DNS
set up.

Yes, this is much longer that usual technique, and requires more effort. But
this is could probably be very well automated to some reasonable extent.

Unfortunately, this is a best practice right now.

_March 2026._
