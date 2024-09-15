#!/usr/bin/env bash

# 1. 将文件复制到容器内部
# sudo docker container cp ./update_hosts.sh jellyfin:/config/

# 2. 检查文件是否已经复制到容器内部
# docker exec jellyfin ls -l /config/update_hosts.sh

# 3. 更新 hosts 命令
# docker exec jellyfin /config/update_hosts.sh

# 4. 查看 hosts 内容
# docker exec jellyfin cat /etc/hosts


if [ -z "$(command -v curl)" -o -z "$(command -v jq)" ]; then
	sed -i 's/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list.d/* \
	&& apt update \
	&& apt install curl jq -y
fi

domain_resolve() {
	DOH_SERVER="1.0.0.1"
	fetch_domain="${1}"
	host_location="${2}"

	sed -i "/${fetch_domain}/d" "${host_location}"

	printf "更新 %-20s 的 ipv4 地址：" "${fetch_domain}"
	echo "# ${fetch_domain} ipv4 resolve by ${DOH_SERVER}" >> "${host_location}"
	ipv4_address=$(curl -s --http2 --header "accept: application/dns-json" "https://${DOH_SERVER}/dns-query?name=${fetch_domain}&type=A" | jq -r '.Answer // empty | .[] | select(.data | test("[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+")) | "\(.data)\t'"${fetch_domain}"'"' || echo "" )
	if [ ! -z "$ipv4_address" ]; then
		update_status="成功"
		echo "$ipv4_address" >> "${host_location}"
	else
		update_status="失败"
	fi
	printf "[ %s ]\n" "${update_status}"

	printf "更新 %-20s 的 ipv6 地址：" "${fetch_domain}"
	echo "# ${fetch_domain} ipv6 resolve by ${DOH_SERVER}" >> "${host_location}"
	ipv6_address=$(curl -s --http2 --header "accept: application/dns-json" "https://${DOH_SERVER}/dns-query?name=${fetch_domain}&type=AAAA" | jq -r '.Answer // empty | .[] | select(.data | contains(":") and (test("^[0-9a-fA-F:]*$"))) | "\(.data)\t'"${fetch_domain}"'"' || echo "" )
	if [ ! -z "$ipv6_address" ]; then
		update_status="成功"
		echo "$ipv6_address" >> "${host_location}"
	else
		update_status="失败"
	fi
	printf "[ %s ]\n" "${update_status}"

}

hosts_source="/etc/hosts"
hosts_parse="/config/hosts"
cp -r "${hosts_source}" "${hosts_parse}"

domain_resolve api.themoviedb.org "${hosts_parse}"
domain_resolve image.tmdb.org "${hosts_parse}"
domain_resolve www.themoviedb.org "${hosts_parse}"

cp -r "${hosts_parse}" "${hosts_source}"
rm "${hosts_parse}"
