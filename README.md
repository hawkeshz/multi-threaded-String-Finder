# string-finder
This GO-lang project implements multi-threaded searching mechanism and focuses on searching a string from given lists(distributed chunks of a single big-file).
The overall application architecture consists of,
# three core components (and corresponding programs): Client, Server and Slaves. 

• The client is responsible for making the request, to the Server, to search the string. 

• The server program on receiving the request, welcomes the client, analyzes and divides the task into parts and allocates these parts to Slaves, which have already registered with the server. 

• The slaves are the workhorses and they register with server and are assigned the tasks. Once they register, they notify the server about the file chunks they have and the server uses this info for scheduling. 


# The Client Program
The client program is responsible for making the request to the Server, to search for the string. It does not take user input but rather uses the command line arguments for the required information from the user. The command line parameters include:

• -searchText=someString – This argument is the compulsory for the user to provide and represent the string that needs to be searched.

• -server=serverIPandPort – This argument represents the IP address and port for the server, to whom the request is being sent.

Once the request has been sent to the server, the client program displays the information to the user and waits for the response from the server. Once the response arrives, it displays it to the user and exits.


# The Server Program
The server is the core component and listens on two different ports; 
It exposes one port to the Clients to receive jobs (string search requests). 
The other port is for Slaves to register themselves with the server. 
Specifically it has following core attributes:

• The ports on which it is listening are provided using command-line arguments and are default to some values if arguments are missing.

• It is a multi-threaded server and is able to handle more than one client and their requests simultaneously.

• It maintains a lit of Slaves to whom it can delegate computation. This list is populated by the messages it receives from the Slaves. List is of course dynamic and changes when slaves enter and leave the system.

• In addition to the list of slaves, it also maintains what data chunks (subfiles) a slave is currently storing. It uses this info for scheduling jobs amongst slaves.

• It also serves as the load-balancer and equally distributes the load amongst Slaves and manages different clients and slaves. The server is the core component.


# The Slave Program
The Slaves are the workhorses and register with the Server, are assigned the tasks and report the success/failure status and associated information to the Server. Specifically the core attributes of the Slave component are as follows:

• The IP for the server and the port is specified by command line arguments as with the Client program.

• On startup, the Slave program connects to server, on the specified IP and port, to register itself with the server. It also provides a list of sub files it is storing.

• Once registered, it listens on a port (whose number can only be provided using command line argument) for the tasks assigned from the server.

• Once the task is assigned, they perform the task and send the response back to the Server.

• The slave quits the search once an "abort" message is received by the server (which signifies that another slave has found the string to be searched).
