# Copyright 2021 Flant CJSC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if bb-is-ubuntu-version? 16.04 ; then

  # bugfix (migration)
  rm -f /etc/apt/sources.list.d/\[trusted\=yes\].list

  if bb-apt-repo? https://apt.flant.ru/common ; then
    exit 0
  fi

  bb-apt-key-add <<END
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mQINBE6C6vABEACaoI0pV5dYfCiLeCzC5SjPzvBhtjRyTgZ7KHR2DVcCtbI2z1WA
tadf8ayF5Gu3TIj72a2zVmJWlehNfNYM/kUJua6EjW3YCx6dttIPussYN6D3I/2u
E3sBDTZqXmf+6DSyiiCBad1HZiFjUqwDSwtOk8sqUixdxlSV6MWdKMYwdZARXTWa
2/sYWHW2vvFmU69IWJkLLxwVADTrYDDbphble4Wk8NLSIGyl+71YJiQ1aMMZq3ug
K2285wYKRG5JR6Zl5UVQnXM12FQOUqB1xOqxwOfidamvtcUvk/i2tAXxlEzZMLyy
d3ixONdYz0EhTvtJdqsOJB6LlWbPSZ+c6RcsehwGi6rU/MeV6aNBBL5kD+rQa6Q4
VyUCzRGqcsjueJ2JzPbwILKWAtjSHoAX6ALqOJ8ig+qpsnaWkg25ecNXGGo1vPCO
TRtkoAUMXzq23ZsNTAIJY/6oUMvvHq5BT8HEWMjy+Gujx70kmNmxmdWCJ7o7eM8R
1YGOWa6j9fWhq12tgI6Hqitsq/eiErwxfjpV3IIIBOMWVF63TgN4Q8KdLqxe87mU
0caYYSAwnW0OZqrO3LvKqMzDZ05Rz0jn485NSv2/+jn1/dcjwUwOlzguyu3QR0Do
3QDRH7q7UgPsDJsXeZWq6gsb4Sm0xWeQlmi1bGrrhQUx6LkbKZWCf9A3AwARAQAB
tDtmbGFudC1hcHQgKEdQRyBrZXkgZm9yIEZsYW50IGFwdCByZXBvc2l0b3J5KSA8
YXB0QGZsYW50LnJ1PokCOAQTAQIAIgUCToLq8AIbAwYLCQgHAwIGFQgCCQoLBBYC
AwECHgECF4AACgkQ5g4WzoYNJVuiSA/9Hot6+2ojL2KI2TdtMTASUyde0WB03+Y6
TNVNas5C3cznyeN05F0ivqdk1G3+ObpB3yi5j/Etzs/HkOXCYIp1wYyo/tJQKFLh
s/4SlYZXNOrdIRWVMGru9CtfWDmb2HmQBMdgcXx9Cj7nfNxeQbXVgAlankAmQFIJ
OUjoK/gmzVNF2Pvu+0EAZppB1lCCYkLtJf8WqSQvQVykvzTu/tNSbPH8HdcqtXHD
AUtKQcQmU4h4CpcijdyyrzOFyL9C5mo8w2L0X5NJAa/vB1RwkaBZ+ddFhEpPSy1D
AxuvBlvgFUGVHk+LCSCwuRLXDypCBtBPH3N3l/3sr+1O5zgGBOJDTydbhXVDkQBv
CBwm4ixirnlAyvc2Rr0U9CRXERCHwYFcfiv3oVcGqBdjblFWGNfvOh4O61IPe2G6
RIMmM3F6uL1pM8n1eCpsataK2KutwgUlAtdXvzGpZ2YjRUs/xkX3Z+ANCskd8y3H
F4iYvytEv+1+jDdBtH44FwuBvHhLVcN7z1VNjKM+pdPZEQKYqYeimfk/c3tQEzb+
2aPq8keGh8HBp8aIhf921Zb8JFqWoUlTKB2p9cPw5hYc8WCewB6hCLj29xezbM3w
aznELJL9fDRE25q2R0TijAdDJ+LYqiq2AO+ls/lnwQRQRAhLJw6MHCkuwurLpss8
oG46JKhRbAW5Ag0EToLq8AEQANyxq/KmvQgKtTj3FFaXxw/G6gmM+SwshMhfO+Nj
eZvX5Dh73jlHve0/nNo/SvNljK4fQW9VOQv34D4XvCjtWGDgBJfv12WcLc5s+Wjg
8xvlNVaM4JUaoVes/XlxYhaObDS0SIpWk2ZQGCJ0ghKzn7NQ7skiDZHr5qgDthPD
ScdwTCgjdojaO3kndqji7KFDPOf1wGaOhTnHP2b7/2AThWIFv+CHo+gtRXwlI3Cx
Vq53w/FLk5iVsd4uUwTbe4XcHrH1rYfxhlKZ13nBNdYTy5ZdpmlPRNR7pbKvbPb/
2cxXwJwGs2WwFjlmkoinxvnC3u2SSToYNuKBkdVgmPb23pMsTdtnJa9jmV4bGFYO
D5o8rOtQP2AFrESXcOVCvwDGuVGlvoQQpqFBiag0BOUCCZfD5ethLzDbBdSqXAnw
0lOv5Nw0hp5UMLRZS9Ln3g4Oh3dRUY1+7v8YHqgQO+apE2NKqRqZ5B4TLAY5tJc4
n2873wAFjpgyw4LvwMoJQkcG7qJSvJF59gHecccgd4Y5TuK4lke5byIYLcgDG/8M
U9IEPLGKJY3j/0nwkowm/NiDer3oT0s7s1067xGq3eWQrGebsN9AaQ4HED1upGEC
EMu2qQyz0iM7LeNcVGtSnA0/Q71l0L3z2Ui+0HVhkHA90g3A5ZIctqF7qv7VLxHD
O9LNABEBAAGJAh8EGAECAAkFAk6C6vACGwwACgkQ5g4WzoYNJVv3Eg/9H0Q2Dsyt
8oO8ZMh7UKA61osF4GwSKOITYitvnf7ZX2Yq/CtKwRfSvLjPGaJnqABMjtDF38w+
bV9lk+6QL8zWav/6/D/ba5Aw1d9a1xNkaiRtZY7QnFvcTH972PuCvGFsVBz8qw8G
I6k8cm9gA0ZsLg6mPIwyjW4CrWNeWqb7hRiMf/djCpsvYXN78pQ1gvqBpGY7kBGa
RaGqyFikCQDWxyAkaWx5Cer1ZkeWhjTQyHjdLOykxEqBBJj4gyLVpqIvFWET6mpS
x/RIvkFNaJ7pUrtyLZX6P3dsESzfuXib1dDfd8b6MM9mcNi07n9sdaoHCih+Zj9Y
rwVHV4rKM5GgSrMphhGitHZwKWdldMbJJuhVFPCIocdox9/bMvbWSv8Qh3YAxyV1
8Pqnbh++LjpjKdCZluAGm+lQWM+eKmF2CiuTiN5vrV7VJWcpgiAQruaGx1FgJ4iA
xhRh8gaotr4bmn4uEWWMOJXsZG26FFkpvjlCDy+gh+34Q8TV2Xa0l6GvMXSKI/0y
R+YrfbsV+plkuuOY3t5wk990Hl2G+zTnmC33nCCOjjRwdNSpVcHBqmLcQgHUxKXj
KWmRmMatXlB/cJQE0eFIv/ggcnxvneHqJ7HYJco9TCNUv08c5f6/RR3HnbUn5jg0
viKjQtPnS8EFyC6zArLgjodpzN2EPQVKp2I=
=85bF
-----END PGP PUBLIC KEY BLOCK-----
END

  bb-apt-repo-add deb https://apt.flant.ru/common xenial main
  bb-apt-repo-add deb-src https://apt.flant.ru/common xenial main
fi
