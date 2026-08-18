package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
)

type hostDTO struct {
	model.Host
	Cluster string   `json:"cluster"`
	Insts   []string `json:"insts"`
}

func (h *H) ListHosts(c *gin.Context) {
	kw := strings.TrimSpace(c.Query("kw"))
	zone := c.Query("zone")
	os := c.Query("os")
	status := c.Query("status")
	page, pageSize := pageParams(c)

	// 全量装载（主机量级小），统计卡基于全量；过滤分页基于条件
	var hosts []model.Host
	h.DB.Order("ip asc").Find(&hosts)
	var clusters []model.Cluster
	h.DB.Find(&clusters)
	clusterName := map[string]string{}
	for _, cl := range clusters {
		clusterName[cl.ID] = cl.Name
	}
	var instances []model.Instance
	h.DB.Find(&instances)
	instsByIP := map[string][]string{}
	for _, inst := range instances {
		if inst.HostIP != "" {
			instsByIP[inst.HostIP] = append(instsByIP[inst.HostIP], inst.Name)
		}
	}

	dtos := make([]hostDTO, 0, len(hosts))
	for _, host := range hosts {
		insts := instsByIP[host.IP]
		if insts == nil {
			insts = []string{}
		}
		dtos = append(dtos, hostDTO{Host: host, Cluster: clusterName[host.ClusterID], Insts: insts})
	}

	filtered := make([]hostDTO, 0, len(dtos))
	for _, hd := range dtos {
		if kw != "" && !strings.Contains(hd.IP+" "+hd.Cluster+" "+strings.Join(hd.Insts, " "), kw) {
			continue
		}
		if zone != "" && hd.Zone != zone {
			continue
		}
		if os != "" && hd.OS != os {
			continue
		}
		if status != "" && hd.Status != status {
			continue
		}
		filtered = append(filtered, hd)
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	// 统计卡：基于全量主机（与原页面行为一致）
	abn, okN := 0, 0
	var sumCpu, sumMem, sumDisk int
	topCpu, topMem, topDisk := dtos[0], dtos[0], dtos[0]
	for _, hd := range dtos {
		if hd.Status != "ok" {
			abn++
		} else {
			okN++
		}
		sumCpu += hd.CPU
		sumMem += hd.Mem
		sumDisk += hd.Disk
		if hd.CPU > topCpu.CPU {
			topCpu = hd
		}
		if hd.Mem > topMem.Mem {
			topMem = hd
		}
		if hd.Disk > topDisk.Disk {
			topDisk = hd
		}
	}
	n := len(dtos)
	if n == 0 {
		n = 1
	}
	top := func(h hostDTO, k string) gin.H { return gin.H{"ip": h.IP, k: map[string]int{"cpu": h.CPU, "mem": h.Mem, "disk": h.Disk}[k]} }

	envelope.OK(c, gin.H{
		"items": filtered[start:end],
		"total": total,
		"stats": gin.H{
			"total": len(dtos), "ok": okN, "abnormal": abn,
			"avgCpu": (sumCpu + n/2) / n, "avgMem": (sumMem + n/2) / n, "avgDisk": (sumDisk + n/2) / n,
			"topCpu": top(topCpu, "cpu"), "topMem": top(topMem, "mem"), "topDisk": top(topDisk, "disk"),
		},
	})
}
