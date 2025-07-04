## PowerVS Image Commands

### 1. capibmadm powervs image import

#### Usage:
Import PowerVS image.

#### Environmental Variable:
IBMCLOUD_API_KEY: IBM Cloud API key.

#### Arguments:
--service-instance-id: PowerVS service instance id.

--bucket: Cloud Object Storage bucket name.

--bucket-region: Cloud Object Storage bucket location.

--object: Cloud Object Storage object name.

--accesskey: Cloud Object Storage HMAC access key.

--secretkey: Cloud Object Storage HMAC secret key.

--name: Name to PowerVS imported image.

--public-bucket: Cloud Object Storage public bucket.

--watch-timeout: watch timeout.

--pvs-storagetype: PowerVS Storage type, accepted values are [tier0, tier1, tier3]..


#### Example:
```shell
export IBMCLOUD_API_KEY=<api-key>
# import image using default storage type (service credential will be autogenerated):
capibmadm powervs image import --service-instance-id <service-instance-id> -b <bucketname> --object rhel-83-10032020.ova.gz --name <imagename> -r <region> --zone <zone>

# import image using default storage type with specifying the accesskey and secretkey explicitly:
capibmadm powervs image import --service-instance-id <service-instance-id> -b <bucketname> --object rhel-83-10032020.ova.gz --name <imagename> -r <region> --zone <zone> --accesskey <accesskey> --secretkey <secretkey>

# with user provided storage type:
capibmadm powervs image import --service-instance-id <service-instance-id> -b <bucketname> --pvs-storagetype <storagetype> --object rhel-83-10032020.ova.gz --name <imagename> -r <region> --zone <zone>

#import image from a public IBM Cloud Storage bucket:
capibmadm powervs image import --service-instance-id <service-instance-id> -b <bucketname>  --object rhel-83-10032020.ova.gz --name <imagename> -r <region> --public-bucket --zone <zone> 

```


### 2. capibmadm powervs image list

#### Usage:
List PowerVS images.

#### Environmental Variable:
IBMCLOUD_API_KEY: IBM Cloud API key.

#### Arguments:
--service-instance-id: PowerVS service instance id.

--zone: PowerVS service instance zone.


#### Example:
```shell
export IBMCLOUD_API_KEY=<api-key>
capibmadm powervs image list --service-instance-id <service-instance-id> --zone <zone>
```