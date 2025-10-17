package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
)

type CurrentPeriod struct {
	Public  PublicFee              `json:"PUBLIC"`
	Members map[string]PersonalFee `json:"-"` // 其餘動態 key 透過 map 接
}

// 最外層
type Data struct {
	CurrentPeriod map[string]json.RawMessage `json:"current_period"`
	Record        map[string]any             `json:"record"`
}

type PersonalFee struct {
	LastPeriodMeterRead float64 `json:"last_period_meter_read"`
	PersonalEletricFee  float64 `json:"personal_electric_fee"`
	TotalEletricFee     float64 `json:"total_electric_fee"`
	WaterFee            float64 `json:"water_fee"`
	NetworkFee          float64 `json:"network_fee"`
	GasFee              float64 `json:"gas_fee"`
	TenatFee            float64 `json:"tenat_fee"`
	TotalFee            float64 `json:"total_fee"`
}

type PublicFee struct {
	Subsidy          float64 `json:"subsidy"`
	WaterFee         float64 `json:"water_fee"`
	PublicEletricFee float64 `json:"public_electric_fee"`
	NetworkFee       float64 `json:"network_fee"`
	GassFee          float64 `json:"gass_fee"`
	Balance          float64 `json:"balance"`
}

func help(Scaner *bufio.ReadWriter) {
	Scaner.WriteString("可用指令：\n")
	Scaner.WriteString("  help                 顯示說明\n")
	Scaner.WriteString("  add -u <user_id>      取得並顯示使用者資料\n")
	Scaner.WriteString("  exit                 離開程式\n\n")
	Scaner.Flush()
}

func init_data() (Data, PublicFee, map[string]PersonalFee, error) {
	b, err := os.ReadFile("data.json")
	if err != nil {
		return Data{}, PublicFee{}, nil, err
	}

	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return Data{}, PublicFee{}, nil, err
	}

	var pub PublicFee
	if err := json.Unmarshal(d.CurrentPeriod["PUBLIC"], &pub); err != nil {
		return Data{}, PublicFee{}, nil, err
	}

	users := make(map[string]PersonalFee, len(d.CurrentPeriod))
	for k, raw := range d.CurrentPeriod {
		if strings.EqualFold(k, "PUBLIC") {
			continue
		}
		if len(raw) == 0 {
			continue
		}
		var pf PersonalFee
		if err := json.Unmarshal(raw, &pf); err != nil {
			return Data{}, PublicFee{}, nil, fmt.Errorf("解析成員 %q 失敗: %w", k, err)
		}
		users[k] = pf
	}

	return d, pub, users, nil
}

func show_info(users map[string]PersonalFee, public PublicFee, Scaner *bufio.ReadWriter) {
	Scaner.WriteString("目前費用如下\n")
	for userName, userInfo := range users {
		Scaner.WriteString(fmt.Sprintf("\t%s :\n", userName))
		Scaner.WriteString(fmt.Sprintf("\t\t應付電費: %.2f\n", userInfo.TotalEletricFee))
		Scaner.WriteString(fmt.Sprintf("\t\t應付水費: %.2f\n", userInfo.WaterFee))
		Scaner.WriteString(fmt.Sprintf("\t\t應付瓦斯費: %.2f\n", userInfo.GasFee))
		Scaner.WriteString(fmt.Sprintf("\t\t應付網路費: %.2f\n", userInfo.NetworkFee))
		Scaner.WriteString(fmt.Sprintf("\t\t總計: %.2f\n\n", userInfo.TotalFee))
	}

	Scaner.WriteString(fmt.Sprintf("\t%s :\n", "public"))
	Scaner.WriteString(fmt.Sprintf("\t\t電費總計: %.2f\n", public.PublicEletricFee))
	Scaner.WriteString(fmt.Sprintf("\t\t水費總計: %.2f\n", public.WaterFee))
	Scaner.WriteString(fmt.Sprintf("\t\t瓦斯費總計: %.2f\n", public.GassFee))
	Scaner.WriteString(fmt.Sprintf("\t\t網路費總計: %.2f\n", public.NetworkFee))
}

func build_data_for_save(prev Data, public PublicFee, users map[string]PersonalFee, Scaner *bufio.ReadWriter) (Data, map[string]json.RawMessage, error) {
	cur := make(map[string]json.RawMessage, len(users)+1)
	show_info(users, public, Scaner)
	// PUBLIC
	pubRaw, err := json.Marshal(public)
	if err != nil {
		return Data{}, nil, fmt.Errorf("marshal PUBLIC: %w", err)
	}
	cur["PUBLIC"] = pubRaw

	// Members
	for name, pf := range users {
		raw, err := json.Marshal(pf)
		if err != nil {
			return Data{}, nil, fmt.Errorf("marshal user %s: %w", name, err)
		}
		cur[name] = raw
	}

	out := Data{
		CurrentPeriod: cur,
		Record:        prev.Record, // 保留舊的 record
	}
	return out, cur, nil
}

func save_data(path string, d Data) error {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func fetch_user_data(user_name string, data Data) (PersonalFee, error) {

	for k, raw := range data.CurrentPeriod {
		if k == user_name {
			var pf PersonalFee
			if err := json.Unmarshal(raw, &pf); err != nil {
				return PersonalFee{}, err
			}
			return pf, nil
		}
	}

	return PersonalFee{}, fmt.Errorf("找不到使用者 %s", user_name)
}

func add_electric_bill(action []string, public PublicFee, users map[string]PersonalFee, Scaner *bufio.ReadWriter) (PublicFee, map[string]PersonalFee) {
	Scaner.WriteString("總用電度數(/kwhs)> ")
	Scaner.Flush()

	total_kwh_str, _ := Scaner.ReadString('\n')
	total_kwh_str = strings.TrimSpace(total_kwh_str)
	total_kwh_flaot, err := strconv.ParseFloat(total_kwh_str, 64)

	if err != nil {
		fmt.Println("輸入錯誤：", err)
		return public, users
	}

	personal_kwh := make(map[string]float64)
	for userName := range users {
		Scaner.WriteString(fmt.Sprintf("%s 本期個人電表度數(/kwhs)> ", userName))
		Scaner.Flush()

		input, _ := Scaner.ReadString('\n')
		input = strings.TrimSpace(input)

		kwh, err := strconv.ParseFloat(input, 64)
		if err != nil {
			Scaner.WriteString("輸入錯誤，請輸入數字\n")
			Scaner.Flush()
			continue
		}

		personal_kwh[userName] = kwh
	}

	Scaner.WriteString("本期平均電價> ")
	Scaner.Flush()
	input, _ := Scaner.ReadString('\n')
	input = strings.TrimSpace(input)

	price, err := strconv.ParseFloat(input, 64)
	if err != nil {
		Scaner.WriteString("輸入錯誤，請輸入數字\n")
		Scaner.Flush()
		return public, users
	}

	Scaner.WriteString("電費計算結果如下:\n")
	Scaner.Flush()
	all_personal_kwh := 0.0
	for userName, userInfo := range users {
		if userName != "PUBLIC" {
			last_kwh := userInfo.LastPeriodMeterRead
			total_price := (personal_kwh[userName] - last_kwh) * price
			Scaner.WriteString(fmt.Sprintf("%s 本期個人用電費用(個人房間):\n\t(%.2f - %.2f) * %.2f = %.2f \n", userName, personal_kwh[userName], last_kwh, price, total_price))
			Scaner.Flush()
			all_personal_kwh += (personal_kwh[userName] - last_kwh)
			userInfo.PersonalEletricFee = total_price
			users[userName] = userInfo
		}
	}

	total_price := (total_kwh_flaot - all_personal_kwh) * price
	Scaner.WriteString(fmt.Sprintf("剩餘公共電度數費用:\n\t(%.2f - %.2f) * %.2f = %.2f \n", total_kwh_flaot, all_personal_kwh, price, total_price))
	Scaner.Flush()
	public.PublicEletricFee = total_price * total_kwh_flaot

	var share_type string
	if len(action) == 1 {
		share_type = "-a"
	} else {
		share_type = action[1]
	}
	switch share_type {
	case "-a": //所有人均攤
		for userName, userInfo := range users {
			if userName == "PUBLIC" {
				continue
			}
			final_fee := userInfo.PersonalEletricFee + (total_price / float64(len(users)))
			Scaner.WriteString(fmt.Sprintf("%s 本期個人用電費用(公共均攤):%.2f + %.2f = %.2f \n", userName, userInfo.PersonalEletricFee, (total_price / float64(len(users))), final_fee))
			Scaner.Flush()

			userInfo.TotalEletricFee = userInfo.PersonalEletricFee + (total_price / float64(len(users)))
			userInfo.TotalFee += userInfo.TotalEletricFee
			userInfo.LastPeriodMeterRead = personal_kwh[userName]
			users[userName] = userInfo
		}
	case "-s": //指定均攤

		// 文字正規化
		for i := 2; i < len(action); i++ {
			action[i] = strings.ToUpper(strings.TrimSpace(action[i]))
		}

		shared_counter := float64(len(action[2:]))
		for userName, userInfo := range users {
			if slices.Contains(action[2:], userName) {
				final_fee := userInfo.PersonalEletricFee + (total_price / shared_counter)
				Scaner.WriteString(fmt.Sprintf("%s 本期個人用電費用(公共均攤):%.2f + %.2f = %.2f \n", userName, userInfo.PersonalEletricFee, (total_price / shared_counter), final_fee))
				Scaner.Flush()
				userInfo.TotalEletricFee = userInfo.PersonalEletricFee + (total_price / shared_counter)
			} else {
				userInfo.TotalEletricFee = userInfo.PersonalEletricFee
			}

			userInfo.TotalFee += userInfo.TotalEletricFee
			userInfo.LastPeriodMeterRead = personal_kwh[userName]
			users[userName] = userInfo
		}
	default:
		fmt.Println("參數錯誤")
		return public, users
	}

	Scaner.WriteString("所有人應付電費如下:\n")
	for userName, userInfo := range users {
		Scaner.WriteString(fmt.Sprintf("\t%s 應付電費: %.2f\n", userName, userInfo.TotalEletricFee))
	}
	Scaner.Flush()

	return public, users
}

func add_water_bill(action []string, public PublicFee, users map[string]PersonalFee, Scaner *bufio.ReadWriter) (PublicFee, map[string]PersonalFee) {
	Scaner.WriteString("總水費 > ")
	Scaner.Flush()

	total_water_str, _ := Scaner.ReadString('\n')
	total_water_str = strings.TrimSpace(total_water_str)
	total_water_flaot, err := strconv.ParseFloat(total_water_str, 64)
	public.WaterFee = total_water_flaot

	if err != nil {
		fmt.Println("輸入錯誤：", err)
		return public, users
	}

	var share_type string
	if len(action) == 1 {
		share_type = "-a"
	} else {
		share_type = action[1]
	}
	switch share_type {
	case "-a": //所有人均攤
		for userName, userInfo := range users {
			if userName == "PUBLIC" {
				continue
			}
			userInfo.WaterFee = total_water_flaot / float64(len(users))
			userInfo.TotalFee += userInfo.WaterFee
			users[userName] = userInfo
		}
	case "-s": //指定均攤

		// 文字正規化
		for i := 2; i < len(action); i++ {
			action[i] = strings.ToUpper(strings.TrimSpace(action[i]))
		}

		shared_counter := float64(len(action[2:]))
		for userName, userInfo := range users {
			if slices.Contains(action[2:], userName) {

				Scaner.Flush()
				userInfo.WaterFee = (total_water_flaot / shared_counter)
				userInfo.TotalFee += userInfo.WaterFee
			} else {
				userInfo.WaterFee = 0
			}
			users[userName] = userInfo
		}
	default:
		fmt.Println("參數錯誤")
		return public, users
	}

	Scaner.WriteString("所有人應付水費如下:\n")
	for userName, userInfo := range users {
		Scaner.WriteString(fmt.Sprintf("\t%s 應付水費: %.2f\n", userName, userInfo.WaterFee))
	}
	Scaner.Flush()

	return public, users
}

func add_gas_bill(action []string, public PublicFee, users map[string]PersonalFee, Scaner *bufio.ReadWriter) (PublicFee, map[string]PersonalFee) {
	Scaner.WriteString("總瓦斯費 > ")
	Scaner.Flush()

	total_gas_str, _ := Scaner.ReadString('\n')
	total_gas_str = strings.TrimSpace(total_gas_str)
	total_gas_flaot, err := strconv.ParseFloat(total_gas_str, 64)
	public.GassFee = total_gas_flaot

	if err != nil {
		fmt.Println("輸入錯誤：", err)
		return public, users
	}

	var share_type string
	if len(action) == 1 {
		share_type = "-a"
	} else {
		share_type = action[1]
	}
	switch share_type {
	case "-a": //所有人均攤
		for userName, userInfo := range users {
			if userName == "PUBLIC" {
				continue
			}
			userInfo.GasFee = total_gas_flaot / float64(len(users))
			userInfo.TotalFee += userInfo.GasFee
			users[userName] = userInfo
		}
	case "-s": //指定均攤

		// 文字正規化
		for i := 2; i < len(action); i++ {
			action[i] = strings.ToUpper(strings.TrimSpace(action[i]))
		}

		shared_counter := float64(len(action[2:]))
		for userName, userInfo := range users {
			if slices.Contains(action[2:], userName) {

				Scaner.Flush()
				userInfo.GasFee = (total_gas_flaot / shared_counter)
			} else {
				userInfo.GasFee = 0
			}
			userInfo.TotalFee += userInfo.GasFee
			users[userName] = userInfo
		}
	default:
		fmt.Println("參數錯誤")
		return public, users
	}

	Scaner.WriteString("所有人應付瓦斯費如下:\n")
	for userName, userInfo := range users {
		Scaner.WriteString(fmt.Sprintf("\t%s 應付瓦斯費: %.2f\n", userName, userInfo.GasFee))
	}
	Scaner.Flush()

	return public, users
}

func add_network_bill(action []string, public PublicFee, users map[string]PersonalFee, Scaner *bufio.ReadWriter) (PublicFee, map[string]PersonalFee) {
	Scaner.WriteString("總網路費 > ")
	Scaner.Flush()

	total_network_str, _ := Scaner.ReadString('\n')
	total_network_str = strings.TrimSpace(total_network_str)
	total_network_flaot, err := strconv.ParseFloat(total_network_str, 64)
	public.NetworkFee = total_network_flaot

	if err != nil {
		fmt.Println("輸入錯誤：", err)
		return public, users
	}

	var share_type string
	if len(action) == 1 {
		share_type = "-a"
	} else {
		share_type = action[1]
	}
	switch share_type {
	case "-a": //所有人均攤
		for userName, userInfo := range users {
			if userName == "PUBLIC" {
				continue
			}
			userInfo.NetworkFee = total_network_flaot / float64(len(users))
			userInfo.TotalFee += userInfo.NetworkFee
			users[userName] = userInfo
		}
	case "-s": //指定均攤

		// 文字正規化
		for i := 2; i < len(action); i++ {
			action[i] = strings.ToUpper(strings.TrimSpace(action[i]))
		}

		shared_counter := float64(len(action[2:]))
		for userName, userInfo := range users {
			if slices.Contains(action[2:], userName) {

				Scaner.Flush()
				userInfo.NetworkFee = (total_network_flaot / shared_counter)
			} else {
				userInfo.NetworkFee = 0
			}
			userInfo.TotalFee += userInfo.NetworkFee
			users[userName] = userInfo
		}
	default:
		fmt.Println("參數錯誤")
		return public, users
	}

	Scaner.WriteString("所有人應付瓦斯費如下:\n")
	for userName, userInfo := range users {
		Scaner.WriteString(fmt.Sprintf("\t%s 應付網路費: %.2f\n", userName, userInfo.NetworkFee))
	}
	Scaner.Flush()

	return public, users
}

func handle_add(action []string, data Data, public PublicFee, users map[string]PersonalFee, Scaner *bufio.ReadWriter) (PublicFee, map[string]PersonalFee) {
	if len(action) < 1 {
		Scaner.WriteString("add 參數錯誤")
		Scaner.Flush()
		help(Scaner)
		return public, users
	}
	switch action[0] {
	case "-u":

		userName := action[1]

		// 正規化資料
		userName = strings.ToUpper(strings.TrimSpace(userName))

		user_info, err := fetch_user_data(userName, data)
		if err != nil {
			Scaner.WriteString(fmt.Sprintf("取得使用者失敗：%v\n", err))
			Scaner.Flush()
			return public, users
		}
		Scaner.WriteString(fmt.Sprintf("成功取得使用者：%v\n", userName))
		users[userName] = user_info
		return public, users
	case "-e":
		public, users = add_electric_bill(action, public, users, Scaner)
		return public, users
	case "-w":
		public, users = add_water_bill(action, public, users, Scaner)
		return public, users
	case "-g":
		public, users = add_gas_bill(action, public, users, Scaner)
		return public, users
	case "-n":
		public, users = add_network_bill(action, public, users, Scaner)
		return public, users
	default:
		return public, users
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	Scaner := bufio.NewReadWriter(reader, writer)
	data, public, users, err := init_data()

	if err != nil {
		fmt.Fprintln(os.Stderr, "讀取輸入錯誤：", err)
		return
	}

	for {
		Scaner.WriteString(`what do you want to do today? (input "help" to get help)> `)
		Scaner.Flush()

		inp, err := Scaner.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Fprintln(os.Stderr, "讀取輸入錯誤：", err)
			continue
		}

		inp = strings.TrimSpace(inp)
		if inp == "" {
			continue
		}

		dirty := false
		action := strings.Fields(inp)
		cmd := action[0]
		switch cmd {
		case "exit", "quit":
			Scaner.WriteString("Bye!\n")
			Scaner.Flush()
			return
		case "help":
			help(Scaner)
		case "add":
			dirty = true
			if action[1] == "-u" {
				dirty = false
			}

			public, users = handle_add(action[1:], data, public, users, Scaner)
		case "info":
			show_info(users, public, Scaner)
		case "update":
			newData, cur, err := build_data_for_save(data, public, users, Scaner)
			if err != nil {
				fmt.Fprintln(os.Stderr, "組裝資料失敗：", err)
				return
			}
			data = newData
			data.CurrentPeriod = cur

			if err := save_data("./data.json", data); err != nil {
				fmt.Fprintln(os.Stderr, "保存失敗：", err)
				return
			}

		default:
			Scaner.WriteString("未知指令，輸入 help 查看說明\n")
			Scaner.Flush()
		}

		if dirty {
			Scaner.WriteString("注意到資料有變動是不是要更新？(y/n)\n")
			Scaner.Flush()

			inp, err := Scaner.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				fmt.Fprintln(os.Stderr, "讀取輸入錯誤：", err)
				continue
			}

			if inp == "y\n" {
				newData, cur, err := build_data_for_save(data, public, users, Scaner)
				if err != nil {
					fmt.Fprintln(os.Stderr, "組裝資料失敗：", err)
					return
				}
				data = newData
				data.CurrentPeriod = cur

				if err := save_data("./data.json", data); err != nil {
					fmt.Fprintln(os.Stderr, "保存失敗：", err)
					return
				}

			}
		}

	}
}
