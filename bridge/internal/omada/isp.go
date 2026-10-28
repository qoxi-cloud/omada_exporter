package omada

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RCooLeR/omada_exporter/internal/model"
	"github.com/RCooLeR/omada_exporter/internal/omada/session"
	log "github.com/rs/zerolog/log"
)

// GetIsp returns cached ISP load data and includes the gateway metadata copied
// from the Open API response for each ISP entry.
func (c *Client) GetIsp() ([]model.Isp, error) {
	return c.session.FetchCached("openapi:isp", c.getIspFresh)
}

// getIspFresh loads ISP dashboard data from the Open API and flattens the
// per-gateway response into a single ISP slice with gateway fields filled in.
func (c *Client) getIspFresh() ([]model.Isp, error) {
	if err := c.requireOpenAPICredentials(); err != nil {
		return nil, err
	}
	omadaCID, siteID := c.ContextIDs()
	url := fmt.Sprintf("%s/openapi/v1/%s/sites/%s/dashboard/gateway/isp/load", c.session.Config.Host, omadaCID, siteID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating GET request for %s", url)
		return nil, err
	}
	resp, err := c.session.MakeOpenApiRequest(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := session.ReadResponseBody(resp, "ISP")
	if err != nil {
		return nil, err
	}
	log.Info().Msg("Received data from ISP endpoint")
	log.Debug().Bytes("data", body).Msg("Received data from ISP endpoint")

	if err := session.ValidateAPIResponse(body, "ISP"); err != nil {
		return nil, err
	}

	ispdata := ispResponse{}
	if err := json.Unmarshal(body, &ispdata); err != nil {
		return nil, err
	}
	var result []model.Isp
	for _, d := range ispdata.Result.Data {
		var rows []model.Isp
		for _, isp := range d.IspInfo.IspArr {
			isp.GatewayName = d.GatewayName
			isp.GatewayMac = d.GatewayMac
			isp.GatewayStatus = d.GatewayStatus
			rows = append(rows, isp)
		}
		// Virtual WANs (Omada 6.x) are only visible on the Web API gateway detail.
		if d.GatewayMac == "" {
			result = append(result, rows...)
			continue
		}
		if detail, err := c.getGatewayDetail(d.GatewayMac); err != nil {
			log.Warn().Err(err).Str("gateway", d.GatewayMac).Msg("ISP: could not read gateway detail for virtual WANs")
		} else {
			rows = mergeVirtualIsps(rows, detail.virtualWans(), d.GatewayName, d.GatewayMac, d.GatewayStatus)
		}
		result = append(result, rows...)
	}

	return result, nil
}

// ispResponse represents the Open API response for ISP data.
type ispResponse struct {
	Result struct {
		Data []struct {
			GatewayName   string `json:"name"`
			GatewayMac    string `json:"mac"`
			GatewayStatus int8   `json:"status"`
			IspInfo       struct {
				IspArr []model.Isp `json:"ispArr"`
			} `json:"ispInfo"`
		} `json:"data"`
	} `json:"result"`
}
