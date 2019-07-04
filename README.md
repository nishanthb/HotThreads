# HotThreads
Bruce Chapman's Hot Threads tool for java

From the site:
http://weblogs.java.net/blog/brucechapman/archive/hotthreads/Main.java/Main.java


Site content (not available any more): (Replace I with Bruce Chapman!)
---
I'm setting up my new desktop development machine, and netbeans installation is atrociously slow, like several minutes just to display the splash screen. The task manager shows the process consuming 50% CPU (on a dual core). After stuffing around barking up several wrong trees I drag out a JMX based tool I wrote a while back to find hot threads in a running application.

I had previously encountered slow startup with Netbeans 5.5 (itself - not the installer) and based on A. Sundararajan's blog Using Mustang's Attach API I had written a tool to output the stack traces for the three busiest threads in a java process.

Here's the source code. And for those that just need to do the same thing without understanding internals, Download the jar file

The program attaches to the local java process (specified by the PID on the command line), it grabs information about the processing time of all threads, twice 500ms apart, and uses that to find the three busiest threads. It then takes 10 stacktrace snapshots of those three threads at 10ms intervals, and looks for the common parts on those stack traces for each thread. If a thread is busy, normally most of the stack stays the same, and just the top part changes. The program then outputs to common parts of the stack traces. From there you can see which thread is running hot, and where it is.
---


Run as  

> java -jar ./HotThread.jar $PID

> /usr/java/default/bin/java  -cp $CP:./HotThread.jar:/usr/java/default/lib/tools.jar  hotthread.Main $PID
