# Superflash


Superflash is a tool for creating images of large disks but only storing the
used blocks. When you then flash those disks, it happens a lot faster
than the old `dd` method (flashing a 64GB Raspberry Pi image, for example is
50x faster)


## Example

First use superflash to create a blank image:

```
superflash blank 31000 mydisk.img
```

Then mount it as a loop device

```
2040  sudo losetup /dev/loop0 mydisk.img
```

Then format it and mount it. You need to pass `nodiscard` to ensure that mkfs
doesn't zero out the whole image

```
sudo mkfs.ext4 -E nodiscard /dev/loop0
mkdir mnt
sudo mount /dev/loop0 mnt
```

Lets create an example file:

```
sudo chmod 777 mnt
head -c 400M /dev/urandom  > mnt/afile.file
```

Now lets unmount the filesystem and create the superflash map (sfmap)

```
sudo umount mnt
superflash encode mydisk.img
```

You should get output like this:

```
end
done. Image was total 32505856000 bytes of which 31438340096 were trimmed
```

The resulting sfmap is far smaller than the img:

```
michael@pandora$ ls -hal
-rw-r--r-- 1 michael michael  31G Jan 16 12:53 mydisk.img
-rw-r--r-- 1 michael michael 430M Jan 16 12:55 mydisk.img.sfmap
```

Instead of doing `dd` you can now flash it like this:

```
superflash flash mydisk.img.sfmap /dev/sdc
```
