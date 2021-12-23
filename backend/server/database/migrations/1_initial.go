package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

var up = `
--
-- PostgreSQL database dump
--

-- Dumped from database version 11.5 (Ubuntu 11.5-0ubuntu0.19.04.1)
-- Dumped by pg_dump version 11.5 (Ubuntu 11.5-0ubuntu0.19.04.1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_with_oids = false;

--
-- Name: sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sessions (
    token text NOT NULL,
    data bytea NOT NULL,
    expiry timestamp with time zone NOT NULL
);


--
-- Name: TABLE sessions; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sessions IS 'Table storing sessions according for scs.';


--
-- Name: system_user; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_user (
    id integer NOT NULL,
    email text NOT NULL,
    lastname text NOT NULL,
    name text NOT NULL,
    password_hash text NOT NULL
);


--
-- Name: TABLE system_user; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.system_user IS 'Table holding a list of users which are known to the system.';


--
-- Name: system_user_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.system_user_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: system_user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.system_user_id_seq OWNED BY public.system_user.id;


--
-- Name: system_user id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user ALTER COLUMN id SET DEFAULT nextval('public.system_user_id_seq'::regclass);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (token);


--
-- Name: system_user system_user_login_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user
    ADD CONSTRAINT system_user_email_unique_idx UNIQUE (email);


--
-- Name: system_user system_user_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user
    ADD CONSTRAINT system_user_pkey PRIMARY KEY (id);


--
-- Name: sessions_expiry_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sessions_expiry_idx ON public.sessions USING btree (expiry);


--
-- PostgreSQL database dump complete
--
`

var down = `
DROP TABLE public.sessions;
DROP TABLE public.system_user;
`

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(up)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(down)
		return err
	})
}
