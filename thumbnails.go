package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/edwvee/exiffix"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/net/context"
)

const fontSize = 75
const SQUARE = false

func transformAntImage(item *AntVideoData, file io.Reader, preview bool) (io.Reader, error) {
	imgIn, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading image: %w", err)
	}
	imgBuffer := bytes.NewReader(imgIn)

	img, _, err := exiffix.Decode(imgBuffer)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	// 1280x720

	height := 720
	if SQUARE {
		height = 1280
	}

	var rgba *image.NRGBA
	if preview {
		rgba = imaging.Fit(img, 1920, 1920, imaging.Lanczos)
	} else {
		rgba = imaging.Fill(img, 1280, height, imaging.Center, imaging.Lanczos)
	}

	if !preview {

		bold, err := getFont("./JosefinSans-Bold.ttf")
		if err != nil {
			return nil, err
		}
		regular, err := getFont("./JosefinSans-Regular.ttf")
		if err != nil {
			return nil, err
		}

		fg := image.White
		c := freetype.NewContext()
		c.SetDPI(72)
		c.SetFont(bold)
		c.SetFontSize(fontSize)
		c.SetClip(rgba.Bounds())
		c.SetDst(rgba)
		c.SetSrc(fg)
		c.SetHinting(font.HintingNone) // font.HintingFull

		// Draw background
		draw.Draw(
			rgba,
			image.Rectangle{
				Min: image.Point{
					X: 810,
					Y: 90,
				},
				Max: image.Point{
					X: rgba.Bounds().Max.X,
					Y: 225,
				},
			},
			image.NewUniform(color.NRGBA{0, 0, 0, 128}),
			image.Point{},
			draw.Over,
		)
		// Draw the text.
		_, err = c.DrawString("Antarctica", freetype.Pt(850, 180))
		if err != nil {
			return nil, fmt.Errorf("drawing font: %w", err)
		}

		if item.Type == "day" {
			c.SetFont(regular)

			// calculate the size of the text by drawing it onto a blank image
			c.SetDst(image.NewRGBA(image.Rect(0, 0, 1280, height)))
			pos, err := c.DrawString(fmt.Sprintf("Day %d: %s", item.Key, item.Short), freetype.Pt(0, 0))
			if err != nil {
				return nil, fmt.Errorf("drawing font: %w", err)
			}

			c.SetDst(rgba)

			draw.Draw(
				rgba,
				image.Rectangle{
					Min: image.Point{
						X: 0,
						Y: height - 220,
					},
					Max: image.Point{
						X: pos.X.Round() + 100,
						Y: height - 85,
					},
				},
				image.NewUniform(color.NRGBA{0, 0, 0, 128}),
				image.Point{},
				draw.Over,
			)

			_, err = c.DrawString(fmt.Sprintf("Day %d: %s", item.Key, item.Short), freetype.Pt(50, height-130))
			if err != nil {
				return nil, fmt.Errorf("drawing font: %w", err)
			}
		}
	}

	r, w := io.Pipe()

	go func() {
		err := jpeg.Encode(w, rgba, nil)
		if err != nil {
			w.CloseWithError(err)
		}
		w.Close()
	}()

	return r, nil
}

func transformGhtImage(item *GhtVideoData, file io.Reader, preview bool) (io.Reader, error) {
	imgIn, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading image: %w", err)
	}
	imgBuffer := bytes.NewReader(imgIn)

	img, _, err := exiffix.Decode(imgBuffer)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	// 1280x720

	height := 720
	if SQUARE {
		height = 1280
	}

	var rgba *image.NRGBA
	if preview {
		rgba = imaging.Fit(img, 1920, 1920, imaging.Lanczos)
	} else {
		rgba = imaging.Fill(img, 1280, height, imaging.Center, imaging.Lanczos)
	}

	if !preview {

		bold, err := getFont("./JosefinSans-Bold.ttf")
		if err != nil {
			return nil, err
		}
		regular, err := getFont("./JosefinSans-Regular.ttf")
		if err != nil {
			return nil, err
		}

		fg := image.White
		c := freetype.NewContext()
		c.SetDPI(72)
		c.SetFont(bold)
		c.SetFontSize(fontSize)
		c.SetClip(rgba.Bounds())
		c.SetDst(rgba)
		c.SetSrc(fg)
		c.SetHinting(font.HintingNone) // font.HintingFull

		// Draw background
		draw.Draw(
			rgba,
			image.Rectangle{
				Min: image.Point{
					X: 280,
					Y: 90,
				},
				Max: image.Point{
					X: rgba.Bounds().Max.X,
					Y: 225,
				},
			},
			image.NewUniform(color.NRGBA{0, 0, 0, 128}),
			image.Point{},
			draw.Over,
		)
		// Draw the text.
		_, err = c.DrawString("The Great Himalaya Trail", freetype.Pt(320, 180))
		if err != nil {
			return nil, fmt.Errorf("drawing font: %w", err)
		}

		if item.Type == "day" {
			c.SetFont(regular)

			// calculate the size of the text by drawing it onto a blank image
			c.SetDst(image.NewRGBA(image.Rect(0, 0, 1280, height)))
			pos, err := c.DrawString(fmt.Sprintf("Day %d: %s", item.Key, item.Short), freetype.Pt(0, 0))
			if err != nil {
				return nil, fmt.Errorf("drawing font: %w", err)
			}

			c.SetDst(rgba)

			draw.Draw(
				rgba,
				image.Rectangle{
					Min: image.Point{
						X: 0,
						Y: height - 220,
					},
					Max: image.Point{
						X: pos.X.Round() + 100,
						Y: height - 85,
					},
				},
				image.NewUniform(color.NRGBA{0, 0, 0, 128}),
				image.Point{},
				draw.Over,
			)

			_, err = c.DrawString(fmt.Sprintf("Day %d: %s", item.Key, item.Short), freetype.Pt(50, height-130))
			if err != nil {
				return nil, fmt.Errorf("drawing font: %w", err)
			}
		}
	}

	r, w := io.Pipe()

	go func() {
		err := jpeg.Encode(w, rgba, nil)
		if err != nil {
			w.CloseWithError(err)
		}
		w.Close()
	}()

	return r, nil
}

func getFont(fname string) (*truetype.Font, error) {
	fontBytes, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, fmt.Errorf("reading font file: %w", err)
	}
	fontParsed, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing font file: %w", err)
	}
	return fontParsed, nil
}

func previewThumbnails(ctx context.Context) error {
	data, err := getAntData()
	if err != nil {
		return fmt.Errorf("can't load days: %w", err)
	}

	files, err := ioutil.ReadDir(thumbnailTestingImportDir)
	if err != nil {
		return fmt.Errorf("getting files in folder: %w", err)
	}

	for _, f := range files {
		matches := filenameRegex.FindStringSubmatch(f.Name())
		if len(matches) != 3 {
			continue
		} else {
			var itemType string
			fileType := matches[1]
			switch fileType {
			case "A":
				itemType = "day"
			}
			keyNumber, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("parsing day number from %q: %w", f.Name, err)
			}
			var item *AntVideoData
			for _, itm := range data {
				if itm.Expedition == "ant" && itm.Type == itemType && itm.Key == keyNumber {
					item = itm
					break
				}
			}
			if item == nil {
				return fmt.Errorf("no item for type %s and key %d for file %q", itemType, keyNumber, f.Name())
			}
			item.ThumbnailTesting = f
		}
	}

	for _, item := range data {

		if item.ThumbnailTesting == nil {
			continue
		}

		fmt.Println("Opening thumbnail", item.Key)
		input, err := os.Open(filepath.Join(thumbnailTestingImportDir, item.ThumbnailTesting.Name()))
		if err != nil {
			return fmt.Errorf("opening thumbnail: %w", err)
		}

		f, err := transformAntImage(item, input, false)
		if err != nil {
			input.Close()
			return fmt.Errorf("transforming thumbnail: %w", err)
		}
		input.Close()

		b, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("reading thumbnail: %w", err)
		}

		err = ioutil.WriteFile(filepath.Join(thumbnailTestingOutputDir, item.ThumbnailTesting.Name()), b, 0666)
		if err != nil {
			return fmt.Errorf("writing thumbnail: %w", err)
		}
		//return nil
	}
	fmt.Print("Done!")
	return nil
}

var ImageFilenames = map[int]string{
	0:   "D000_kjrsmq",
	1:   "D001_d3aqux",
	2:   "D002_ensw2l",
	3:   "D003_lenjhr",
	4:   "D004_rzp9ji",
	5:   "D005_x0ttun",
	7:   "D007_hd7stk",
	8:   "D008_xsfdtb",
	10:  "D010_bf8frq",
	12:  "D012_wqchhz",
	13:  "D013_w37mzv",
	14:  "D014_p8tfdf",
	16:  "D016_m8azbl",
	17:  "D017_uezxy2",
	18:  "D018_svyjec",
	21:  "D021_xk5cn8",
	22:  "D022_zcsvar",
	23:  "D023_qb4k3b",
	24:  "D024_lruol3",
	25:  "D025_chhio7",
	27:  "D027_oqyal5",
	28:  "D028_xlwkki",
	29:  "D029_pgzbnx",
	30:  "D030_ab8mjr",
	31:  "D031_cwobfv",
	32:  "D032_fxnmbg",
	33:  "D033_ssdhof",
	34:  "D034_lhtsf6",
	36:  "D036_qpa56j",
	37:  "D037_apcw8f",
	38:  "D038_dfv82b",
	39:  "D039_tuvnen",
	40:  "D040_xnpiwi",
	41:  "D041_bfs1ib",
	42:  "D042_fzf0ho",
	43:  "D043_qx8ehk",
	45:  "D045_lwgaah",
	46:  "D046_ufkkbi",
	47:  "D047_lqagmx",
	48:  "D048_wpqulq",
	49:  "D049_cvgpfg",
	50:  "D050_zkbt42",
	51:  "D051_lv0cio",
	53:  "D053_hbdgfo",
	54:  "D054_ecnrkl",
	55:  "D055_yrw2xm",
	56:  "D056_w4wf9m",
	57:  "D057_ymcuea",
	58:  "D058_se1hx4",
	60:  "D060_izjq4w",
	61:  "D061_insmbf",
	62:  "D062_lfyfp1",
	63:  "D063_batldh",
	64:  "D064_dqsrrs",
	65:  "D065_l4xyye",
	67:  "D067_lmfrfz",
	68:  "D068_res07o",
	69:  "D069_bptvei",
	71:  "D071_mml8ov",
	72:  "D072_rexlvf",
	73:  "D073_pexqvu",
	74:  "D074_icgbae",
	75:  "D075_mdnd26",
	76:  "D076_shdxq7",
	77:  "D077_v3ndmq",
	78:  "D078_auwoc0",
	79:  "D079_dcwcpg",
	80:  "D080_rzewsy",
	82:  "D082_frbpbt",
	83:  "D083_rdcbcb",
	88:  "D088_qmdikx",
	89:  "D089_c46wnv",
	90:  "D090_mdbqp6",
	91:  "D091_gwzswk",
	92:  "D092_zwupjz",
	95:  "D095_ypzrre",
	97:  "D097_jksz9p",
	98:  "D098_jvqfqr",
	100: "D100_wbjc7g",
	101: "D101_snelus",
	103: "D103_sucoqr",
	104: "D104_z3comm",
	105: "D105_hkmrwz",
	106: "D106_vz1dp1",
	107: "D107_nw5wqw",
	108: "D108_y2nvtq",
	109: "D109_ogw037",
	110: "D110_dpsyau",
	111: "D111_xkxvii",
	112: "D112_q1wjtc",
	113: "D113_tvkrun",
	115: "D115_ycrr7l",
	116: "D116_f2wfwa",
	117: "D117_jqdbgu",
	118: "D118_unualg",
	120: "D120_xctule",
	121: "D121_g8ibft",
	122: "D122_pandz6",
	123: "D123_hixsr1",
	124: "D124_zlug8h",
	125: "D125_yostdw",
	126: "D126_ujsoeq",
	127: "D127_yas0ak",
	129: "D129_qghqmk",
	130: "D130_l1gauw",
	131: "D131_nemfzs",
	132: "D132_ww4emo",
	133: "D133_tb9wgk",
	134: "D134_vptfgb",
	136: "D136_wbnix6",
	137: "D137_rmqzrj",
	138: "D138_ovqusj",
	139: "D139_vgdiy1",
	140: "D140_cgbnz9",
	141: "D141_kj0qmc",
	142: "D142_k8rnhn",
	143: "D143_lgs7js",
	145: "D145_gprqqp",
	146: "D146_ksbto3",
	147: "D147_dudzml",
	148: "D148_tqxxn5",
	149: "D149_i7xlm1",
	150: "D150_do2pb9",
	152: "D152_imxpdf",
	153: "D153_utf5cg",
	154: "D154_yiiy2b",
}

var imageFilenamesNoText = map[int]string{
	1:   "D001_mu7e02",
	2:   "D002_kpz0qa",
	3:   "D003_ecut84",
	4:   "D004_wq7mlt",
	5:   "D005_o3lz5y",
	7:   "D007_fxniig",
	8:   "D008_pttufv",
	10:  "D010_zqsnfi",
	12:  "D012_py87ov",
	13:  "D013_pag5yl",
	14:  "D014_zqvjxd",
	16:  "D016_kxcm0y",
	17:  "D017_yvpf4x",
	18:  "D018_mnsuoj",
	21:  "D021_eom8qg",
	22:  "D022_n8jsxl",
	23:  "D023_bbsc02",
	24:  "D024_wnkpep",
	25:  "D025_itsal9",
	27:  "D027_siruo6",
	28:  "D028_ljdqml",
	29:  "D029_vncgix",
	30:  "D030_z8keod",
	31:  "D031_rfijnp",
	32:  "D032_txfmyg",
	33:  "D033_otozka",
	34:  "D034_n0vaoi",
	36:  "D036_qt0edv",
	37:  "D037_rxryye",
	38:  "D038_w2yyqm",
	39:  "D039_gsv2d3",
	40:  "D040_aum2rl",
	41:  "D041_xqfsur",
	42:  "D042_nmogbx",
	43:  "D043_x8iulf",
	45:  "D045_gunwmc",
	46:  "D046_lpkufh",
	47:  "D047_rc2ehz",
	48:  "D048_ymadff",
	49:  "D049_geloif",
	50:  "D050_dneuov",
	51:  "D051_ckk1fp",
	53:  "D053_dhdm2k",
	54:  "D054_v6ed28",
	55:  "D055_ishorw",
	56:  "D056_z1m7ml",
	57:  "D057_o9gwaw",
	58:  "D058_u1dsve",
	60:  "D060_xgb3je",
	61:  "D061_fi5dov",
	62:  "D062_lt1xlk",
	63:  "D063_oumtz2",
	64:  "D064_d4pjji",
	65:  "D065_elyfhy",
	67:  "D067_a4cuqz",
	68:  "D068_hg9gdb",
	69:  "D069_ewntx6",
	71:  "D071_iah2mz",
	72:  "D072_tmkb6n",
	73:  "D073_xmqte9",
	74:  "D074_ytyfcg",
	75:  "D075_ky40tc",
	76:  "D076_afypus",
	77:  "D077_d4pmea",
	78:  "D078_sa8okk",
	79:  "D079_lsjgno",
	80:  "D080_w75olh",
	82:  "D082_nz8guz",
	83:  "D083_dwdeoa",
	88:  "D088_y1x8u4",
	89:  "D089_i6tihc",
	90:  "D090_pltrqj",
	91:  "D091_bc3joa",
	92:  "D092_vtjycm",
	95:  "D095_hxmkug",
	97:  "D097_jqwk7c",
	98:  "D098_mtrmwz",
	100: "D100_waasnf",
	101: "D101_k0uryi",
	103: "D103_yup1wk",
	104: "D104_d80ijs",
	105: "D105_mvqsiu",
	106: "D106_jx8bx1",
	107: "D107_jh5krn",
	108: "D108_bzhfsq",
	109: "D109_wnwymz",
	110: "D110_y1ywuj",
	111: "D111_dkbjtd",
	112: "D112_dr0nwm",
	113: "D113_ulop5w",
	115: "D115_stppqo",
	116: "D116_ppemed",
	117: "D117_hwnflh",
	118: "D118_tpx08x",
	120: "D120_j3gcun",
	121: "D121_puszmu",
	122: "D122_ecsrse",
	123: "D123_uvlzjh",
	124: "D124_hnfoer",
	125: "D125_naabvr",
	126: "D126_yoaarh",
	127: "D127_a5isaj",
	129: "D129_xtt465",
	130: "D130_gt3lac",
	131: "D131_dpnqyw",
	132: "D132_txy67n",
	133: "D133_h7kyej",
	134: "D134_fgfslu",
	136: "D136_omzu6o",
	137: "D137_pib7tx",
	138: "D138_xxsxyh",
	139: "D139_cmzpac",
	140: "D140_j3rhz0",
	141: "D141_lcejdk",
	142: "D142_qcnggk",
	143: "D143_duuxmr",
	145: "D145_y6fmof",
	146: "D146_kxm2nd",
	147: "D147_smjy8k",
	148: "D148_lie6tn",
	149: "D149_pautnn",
	150: "D150_ix30eu",
	152: "D152_hrcd97",
	153: "D153_csuifi",
	154: "D154_abqutm",
}
