echo 'tail -f out0.txt | grep ^[^2]'
factomd -count=16 -port="8091" -networkPort="34340" -logPort="6061" -peers="127.0.0.1:34341" -network=LOCAL -blktime=60 -net=alot+ -startdelay=20 > out0.txt

