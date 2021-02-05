Proxy3270, a 3270 forwarding service
====================================

[Proxy3270](https://github.com/racingmars/proxy3270) allows a user to open a 3270 session to the server and choose from a list of other 3270 servers to connect to. This allows you to have several services in your network and provide one point of access--the Proxy3270 service. Proxy3270 supports TLS connects to itself and to the hosts it connects to.

I have some plans for the future, but at the moment the service is very simple: it is statically configured from a JSON configuration file, and it will allow the user to choose a service to connect to, then disconnect when/if the remote server session is done.

Usage
-----

Build the project with the usual `go build`, resulting in the proxy3270 binary. Create a configuration file (see `config.sample.json`) named `config.json` with your hosts -- right now, you may have up to 26 hosts and the names may be up to 30 characters long. Then, run proxy3270. By default, it will listen on port 3270. You may also use a few flags:

 - `-port <port>` set the port number to listen on. (Default 3270)
 - `-debug` enable debug logging level.
 - `-debug3270` enable debug output in the go3270 library.
 - `-trace` enable trace logging level (logs all data received from clients and servers during forwarding).
 - `-config <file>` use a config file other than config.json.
 - `-telnetTimeout <seconds>` set the time to wait for 3270 client response during "un-negotiation" before forwarding to remote host. The default of 1 second should be fine in most cases, but if using IBM PCOMM, I need to set this to 5 seconds.

To enable the TLS listener:

 - `-enabletls` Enables the TLS listener.
 - `-pubkey <filename>` PEM-encoded X.509 certificate for this server, with optional intermediate bundle after the server certificate. (Default pubkey.pem)
 - `-privkey <filename>` Unencrypted private key for the certificate in the public key file. (Default privkey.pem)
 - `-tlsport <port>` Port number for the TLS listener. (Default 4270)

Limitations
-----------

(Some of these will change in the future)

 - You may have up to 999 hosts in your configuration file.
 - Server names are limited to 65 characters.
 - To change the configuration, you must restart the server (which will drop all active connections).

Other Notes
-----------

At first I tried to "recover" the 3270 session after the remote server closes the connection, and present the menu to the user again to initiate a new menu. For reasons I haven't solved yet, that isn't working: I'll need to look at some packet captures to see what's happening on the wire. I'd like to solve this if it's possible...I can't think of why it wouldn't be, unless doing something like a "logoff" from the z/OS network solicitor is sending some sequence of telnet or 3270 commands which put the client in a "done" state that cannot be reversed.

My larger plans for this application are to make the configuration database-backed (just with an embedded Go database library so there are no external database dependencies) to support dynamic configuration from administration screens within the application, as well as add user authentication to support different users having access to different hosts. If you'd like to contribute, please feel free!

Acknowledgements
----------------

[Moshix](https://github.com/moshix) suggested the idea originally, and it fit in well with something that I wanted to do which this may evolve into.

I'm using my own [go3270 library](https://github.com/racingmars/go3270/) for screen control during the menu selection before proxying traffic to the selected host.

Logging is provided by [zerolog](https://github.com/rs/zerolog).

License
-------

Copyright 2020 by Matthew R. Wilson (mwilson@mattwilson.org)

proxy3270 is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

proxy3270 is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with proxy3270. If not, see <https://www.gnu.org/licenses/>.