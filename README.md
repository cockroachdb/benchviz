Benchviz is a tool that hooks up to an s3 instance and deploys daily benchmark stats as a JSON file to be visualized.

#USAGE

1. Setup an AWS account hooked up with S3.
2. Install the aws command line tool.
3. Setup aws to be connected with your account.
4. Set the following environment variables:

>AWSBUCKETNAME=\<name of your aws bucket\><br/>
>BENCHDEPLOY=\<full path to s3 mirror\><br/>
>BENCHSAMPLES=\<full path to your bench stat data\>
>
        
##Benchsamples

Benchsamples is a directory that contains folders filled with historical benchmark results. The format of the directory (via example) is

/benchSamples<br/>
../01-01-2016<br/>
../02-01-2016<br/>
..../cockroach<br/>
....../kv<br/>
......../kv.test.stdout<br/>
....../sql<br/>
......../parser<br/>
........../parser.test.stdout<br/>
....../roachpb<br/>
....../storage<br/>
....../util<br/>
../03--01-2016<br/>

Important note: the directory must be named the date of the commit where the results were received from in DD-MM-YYYY format. In addition, the directory must store the results of benchmark tests in the directories that have benchmark tests under the name \<directory\>.test.stdout. This directory hierarchy is designed in this way because this is the format that we have our current historical benchmark data in. 